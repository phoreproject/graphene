package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/execution"
	"github.com/phoreproject/synapse/shard/mempool"
	"github.com/phoreproject/synapse/shard/transfer"
	"github.com/phoreproject/synapse/utils"
	"github.com/prysmaticlabs/go-ssz"
)

// ShardChainInitializationParameters are the initialization parameters from the crosslink.
type ShardChainInitializationParameters struct {
	RootBlockHash chainhash.Hash
	RootSlot      uint64
	GenesisTime   uint64
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
	Mempool                  *mempool.ShardMempool
	StateManager             *execution.BasicFullStateManager
}

// NewShardManager initializes a new shard manager responsible for keeping track of a shard chain.
func NewShardManager(shardID uint64, init ShardChainInitializationParameters, beaconClient pb.BlockchainRPCClient) *ShardManager {

	genesisBlock := primitives.GetGenesisBlockForShard(shardID)

	return &ShardManager{
		ShardID:                  shardID,
		Chain:                    NewShardChain(init.RootSlot, &genesisBlock),
		Index:                    NewShardBlockIndex(genesisBlock),
		InitializationParameters: init,
		BeaconClient:             beaconClient,
		Mempool:                  mempool.NewShardMempool(mempool.ValidateTrue, mempool.PrioritizeEqual),
		StateManager:             execution.NewBasicFullStateManager(transfer.Code, uint32(shardID)), // TODO: this should be loaded dynamically instead of directly from the filesystem
	}
}

// SubmitTransaction submits a transaction to the shard.
func (sm *ShardManager) SubmitTransaction(transaction []byte) error {
	return sm.Mempool.SubmitTransaction(transaction)
}

// SubmitBlock submits a block to the chain for processing.
func (sm *ShardManager) SubmitBlock(block primitives.ShardBlock) error {
	// first, let's make sure we have the parent block
	hasParent := sm.Index.HasBlock(block.Header.PreviousBlockHash)
	if !hasParent {
		return fmt.Errorf("missing parent block %s", block.Header.PreviousBlockHash)
	}

	parentStateRoot, _ := sm.Index.GetNodeByHash(&block.Header.PreviousBlockHash)

	transactions := make([][]byte, len(block.Body.Transactions))
	for i := range transactions {
		transactions[i] = block.Body.Transactions[i].TransactionData
	}

	// update state based on block transactions
	newStateRoot, err := sm.StateManager.CheckTransition(parentStateRoot.StateRoot, transactions)
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

	_, err = sm.StateManager.Transition(parentStateRoot.StateRoot, transactions)
	if err != nil {
		return err
	}

	node, err := sm.Index.AddToIndex(block)
	if err != nil {
		return err
	}

	// TODO: improve fork choice here
	tipNode, err := sm.Chain.Tip()
	if err != nil {
		return err
	}

	if node.Height > tipNode.Height || (node.Height == tipNode.Height && node.Slot > tipNode.Slot) {
		sm.Chain.SetTip(node)

		sm.Mempool.RemoveTransactionsFromBlock(&block)
	}

	return nil
}
