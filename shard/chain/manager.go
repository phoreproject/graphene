package chain

import (
	"context"
	"errors"
	"fmt"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/db"
	"github.com/phoreproject/synapse/shard/state"
	"github.com/phoreproject/synapse/shard/transfer"
	"github.com/phoreproject/synapse/utils"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/sirupsen/logrus"
	"sync"
)

// ShardChainInitializationParameters are the initialization parameters from the crosslink.
type ShardChainInitializationParameters struct {
	RootBlockHash chainhash.Hash
	RootSlot      uint64
	GenesisTime   uint64
}

// ShardChainActionNotifee is a notifee for any shard chain actions.
type ShardChainActionNotifee interface {
	AddBlock(block *primitives.ShardBlock, newTip bool)
	FinalizeBlockHash(blockHash chainhash.Hash, slot uint64)
}

// ShardManager represents part of the blockchain on a specific shard (starting from a specific crosslinked block hash),
// and continued to a certain point.
type ShardManager struct {
	ShardID                  uint64
	Chain                    *ShardChain
	Index                    *ShardBlockIndex
	InitializationParameters ShardChainInitializationParameters
	BeaconClient             pb.BlockchainRPCClient
	Config                   config.Config
	BlockDB                  db.ShardBlockDatabase
	SyncManager              *ShardSyncManager

	stateManager *state.ShardStateManager
	shardInfo    state.ShardInfo

	notifees     []ShardChainActionNotifee
	notifeesLock sync.Mutex
}

// NewShardManager initializes a new shard manager responsible for keeping track of a shard chain.
func NewShardManager(shardID uint64, init ShardChainInitializationParameters, beaconClient pb.BlockchainRPCClient, hn *p2p.HostNode) (*ShardManager, error) {

	genesisBlock := primitives.GetGenesisBlockForShard(shardID)

	stateDB := csmt.NewInMemoryTreeDB()
	shardInfo := state.ShardInfo{
		CurrentCode: transfer.Code, // TODO: this should be loaded dynamically instead of directly from the filesystem
		ShardID:     uint32(shardID),
	}


	sm := &ShardManager{
		ShardID:                  shardID,
		Chain:                    NewShardChain(init.RootSlot, &genesisBlock),
		Index:                    NewShardBlockIndex(genesisBlock),
		InitializationParameters: init,
		BeaconClient:             beaconClient,
		shardInfo:                shardInfo,
		stateManager:             state.NewShardStateManager(stateDB, init.RootSlot, init.RootBlockHash, shardInfo),
		notifees:                 []ShardChainActionNotifee{},
		BlockDB:                  db.NewMemoryBlockDB(),
	}

	syncManager, err := NewShardSyncManager(hn, sm, shardID)
	if err != nil {
		return nil, err
	}

	sm.SyncManager = syncManager

	sm.ListenForNewCrosslinks()

	return sm, nil
}

// RegisterNotifee registers a notifee for shard actions
func (sm *ShardManager) RegisterNotifee(n ShardChainActionNotifee) {
	sm.notifeesLock.Lock()
	defer sm.notifeesLock.Unlock()
	sm.notifees = append(sm.notifees, n)
}

// UnregisterNotifee unregisters a notifee for shard actions
func (sm *ShardManager) UnregisterNotifee(n ShardChainActionNotifee) {
	sm.notifeesLock.Lock()
	defer sm.notifeesLock.Unlock()
	for i, other := range sm.notifees {
		if other == n {
			sm.notifees = append(sm.notifees[:i], sm.notifees[i+1:]...)
		}
	}
}

// ListenForNewCrosslinks listens for new crosslinks.
func (sm *ShardManager) ListenForNewCrosslinks() {
	go func() {
		stream, err := sm.BeaconClient.CrosslinkStream(context.Background(), &pb.CrosslinkStreamRequest{
			ShardID: uint64(sm.shardInfo.ShardID),
		})
		if err != nil {
			logrus.Error(err)
		}

		for {
			newCrosslink, err := stream.Recv()
			if err != nil {
				logrus.Error(err)
				break
			}

			crosslinkHash, err := chainhash.NewHash(newCrosslink.BlockHash)
			if err != nil {
				logrus.Error(err)
				break
			}

			for _, n := range sm.notifees {
				n.FinalizeBlockHash(*crosslinkHash, newCrosslink.Slot)
			}

			if err := sm.stateManager.Finalize(*crosslinkHash, newCrosslink.Slot); err != nil {
				logrus.Error(err, sm.InitializationParameters.RootBlockHash)
				break
			}
			// TODO: also update fork choice with new crosslink
		}
	}()
}

// GetKey gets a key from the current state.
func (sm *ShardManager) GetKey(key chainhash.Hash) (*chainhash.Hash, error) {
	var out *chainhash.Hash
	err := sm.stateManager.GetTip().View(func(tx csmt.TreeDatabaseTransaction) error {
		val, err := tx.Get(key)
		if err != nil {
			return err
		}
		out = val
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ProcessBlock submits a block to the chain for processing.
func (sm *ShardManager) ProcessBlock(block primitives.ShardBlock) error {
	h, _ := ssz.HashTreeRoot(block)

	logrus.WithFields(logrus.Fields{
		"slot":  block.Header.Slot,
		"hash":  chainhash.Hash(h),
		"shard": sm.ShardID,
		"transactions": len(block.Body.Transactions),
	}).Debug("processing block")

	// first, let's make sure we have the parent block
	hasParent := sm.Index.HasBlock(block.Header.PreviousBlockHash)
	if !hasParent {
		return fmt.Errorf("missing parent block %s", block.Header.PreviousBlockHash)
	}

	if sm.stateManager.Has(h) {
		return nil
	}

	newStateRoot, err := sm.stateManager.Add(&block)
	if err != nil {
		return err
	}

	if !block.Header.StateRoot.IsEqual(newStateRoot) {
		return fmt.Errorf("expected new state root: %s, but block state root is %s", newStateRoot, block.Header.StateRoot)
	}

	if block.Header.Slot*uint64(sm.Config.SlotDuration)+sm.InitializationParameters.GenesisTime > uint64(utils.Now().Unix()) {
		return fmt.Errorf("shard block submitted too soon")
	}

	// we should check the signature against the beacon chain
	proposer, err := sm.BeaconClient.GetShardProposerForSlot(context.Background(), &pb.GetShardProposerRequest{
		ShardID: sm.ShardID,
		Slot:    block.Header.Slot,
	})

	if err != nil {
		return err
	}

	if len(proposer.ProposerPublicKey) > 96 {
		return errors.New("public key returned from beacon chain is incorrect")
	}

	blockWithNoSignature := block.Copy()
	blockWithNoSignature.Header.Signature = bls.EmptySignature.Serialize()

	blockHashNoSignature, err := ssz.HashTreeRoot(blockWithNoSignature)
	if err != nil {
		return err
	}

	var pubkeySerialized [96]byte
	copy(pubkeySerialized[:], proposer.ProposerPublicKey)

	proposerPublicKey, err := bls.DeserializePublicKey(pubkeySerialized)
	if err != nil {
		return err
	}

	blockSig, err := bls.DeserializeSignature(block.Header.Signature)
	if err != nil {
		return err
	}

	valid, err := bls.VerifySig(proposerPublicKey, blockHashNoSignature[:], blockSig, bls.DomainShardProposal)
	if err != nil {
		return err
	}

	if !valid {
		return errors.New("block signature was not valid")
	}

	node, err := sm.Index.AddToIndex(block)
	if err != nil {
		return err
	}

	if err := sm.BlockDB.SetBlock(&block); err != nil {
		return err
	}

	// TODO: improve fork choice here
	tipNode, err := sm.Chain.Tip()
	if err != nil {
		return err
	}

	newTip := false
	if node.Height > tipNode.Height || (node.Height == tipNode.Height && node.Slot > tipNode.Slot) {
		sm.Chain.SetTip(node)
		if err := sm.stateManager.SetTip(node.BlockHash); err != nil {
			return err
		}

		newTip = true
	}

	for _, n := range sm.notifees {
		n.AddBlock(&block, newTip)
	}

	return nil
}
