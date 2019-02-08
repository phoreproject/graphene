package beacon

import (
	"fmt"
	"sync"

	"github.com/phoreproject/prysm/shared/ssz"

	"github.com/phoreproject/synapse/bls"
	logger "github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/beacon/primitives"

	"github.com/phoreproject/synapse/chainhash"
)

var zeroHash = chainhash.Hash{}

type blockNode struct {
	hash   chainhash.Hash
	height uint64
	parent *blockNode
}

type blockchainView struct {
	tip   *blockNode
	chain []*blockNode
	lock  *sync.Mutex
}

func (bv *blockchainView) GetBlock(n int) (*blockNode, error) {
	bv.lock.Lock()
	defer bv.lock.Unlock()
	if len(bv.chain) > n && n >= 0 {
		return bv.chain[n], nil
	}
	return nil, fmt.Errorf("block %d does not exist", n)
}

func (bv *blockchainView) Height() int {
	bv.lock.Lock()
	defer bv.lock.Unlock()
	return len(bv.chain) - 1
}

// Blockchain represents a chain of blocks.
type Blockchain struct {
	chain     blockchainView
	db        db.Database
	config    *config.Config
	state     primitives.State
	stateLock *sync.Mutex
}

// NewBlockchainWithInitialValidators creates a new blockchain with the specified
// initial validators.
func NewBlockchainWithInitialValidators(db db.Database, config *config.Config, validators []InitialValidatorEntry) (*Blockchain, error) {
	b := &Blockchain{
		db:     db,
		config: config,
		chain: blockchainView{
			chain: []*blockNode{},
			lock:  &sync.Mutex{},
		},
		stateLock: &sync.Mutex{},
	}
	err := b.InitializeState(validators, 0)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// InitialValidatorEntry is the validator entry to be added
// at the beginning of a blockchain.
type InitialValidatorEntry struct {
	PubKey                bls.PublicKey
	ProofOfPossession     bls.Signature
	WithdrawalShard       uint32
	WithdrawalCredentials chainhash.Hash
	RandaoCommitment      chainhash.Hash
	PoCCommitment         chainhash.Hash
	DepositSize           uint64
}

const (
	// RoleProposer is assigned to validators who need to propose a shard block.
	RoleProposer = iota

	// RoleAttester is assigned to validators who need to attest to a shard block.
	RoleAttester
)

const (
	// InitialForkVersion is version #1 of the chain.
	InitialForkVersion = iota
)

// UpdateChainHead updates the blockchain head if needed
func (b *Blockchain) UpdateChainHead(n *primitives.Block) error {
	b.chain.lock.Lock()
	// TODO
	if int64(n.BlockHeader.SlotNumber) > int64(len(b.chain.chain)-1) {
		blockHash, err := ssz.TreeHash(n)
		if err != nil {
			b.chain.lock.Unlock()
			return err
		}
		logger.WithField("hash", blockHash).WithField("height", n.BlockHeader.SlotNumber).Debug("updating blockchain head")
		b.chain.lock.Unlock()
		err = b.SetTip(blockNode{
			hash:   blockHash,
			height: n.BlockHeader.SlotNumber,
			parent: b.chain.tip,
		})
		if err != nil {
			return err
		}
	} else {
		b.chain.lock.Unlock()
	}
	return nil
}

// SetTip sets the tip of the chain.
func (b *Blockchain) SetTip(bl blockNode) error {
	b.chain.lock.Lock()
	defer b.chain.lock.Unlock()
	needed := bl.height
	if uint64(cap(b.chain.chain)) < needed {
		nodes := make([]*blockNode, needed, needed+100)
		copy(nodes, b.chain.chain)
		b.chain.chain = nodes
	} else {
		b.chain.chain = b.chain.chain[0:needed]
	}

	return nil
}

// Tip returns the block at the tip of the chain.
func (b Blockchain) Tip() chainhash.Hash {
	b.chain.lock.Lock()
	tip := b.chain.tip.hash
	b.chain.lock.Unlock()
	return tip
}

// GetNodeByHeight gets a node from the active blockchain by height.
func (b Blockchain) getNodeByHeight(height uint64) (*blockNode, error) {
	b.chain.lock.Lock()
	defer b.chain.lock.Unlock()
	if uint64(len(b.chain.chain)-1) < height {
		return nil, fmt.Errorf("attempted to retrieve block hash of height > chain height")
	}
	node := b.chain.chain[height]
	return node, nil
}

// GetHashByHeight gets the block hash at a certain height.
func (b Blockchain) GetHashByHeight(height uint64) (chainhash.Hash, error) {
	b.chain.lock.Lock()
	defer b.chain.lock.Unlock()
	if uint64(len(b.chain.chain)-1) < height {
		return chainhash.Hash{}, fmt.Errorf("attempted to retrieve block hash of height > chain height")
	}
	node := b.chain.chain[height]
	return node.hash, nil
}

// Height returns the height of the chain.
func (b *Blockchain) Height() int {
	return b.chain.Height()
}

// LastBlock gets the last block in the chain
func (b *Blockchain) LastBlock() (*primitives.Block, error) {
	bl, err := b.chain.GetBlock(b.chain.Height())
	if err != nil {
		return nil, err
	}
	block, err := b.db.GetBlockForHash(bl.hash)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GetConfig returns the config used by this blockchain
func (b *Blockchain) GetConfig() *config.Config {
	return b.config
}

// GetSlotAndShardAssignment gets the shard and slot assignment for a specific
// validator.
func (b *Blockchain) GetSlotAndShardAssignment(validatorID uint32) (uint64, uint64, int, error) {
	earliestSlotInArray := b.state.Slot%b.config.EpochLength - b.config.EpochLength
	if earliestSlotInArray < 0 {
		earliestSlotInArray = 0
	}
	for i, slot := range b.state.ShardAndCommitteeForSlots {
		for j, committee := range slot {
			for v, validator := range committee.Committee {
				if uint32(validator) != validatorID {
					continue
				}
				if j == 0 && v == i%len(committee.Committee) {
					return committee.Shard, uint64(i) + earliestSlotInArray, RoleProposer, nil
				}
				return committee.Shard, uint64(i) + earliestSlotInArray, RoleAttester, nil
			}
		}
	}
	return 0, 0, 0, fmt.Errorf("validator not found in set %d", validatorID)
}

// GetValidatorAtIndex gets the validator at index
func (b *Blockchain) GetValidatorAtIndex(index uint32) (*primitives.Validator, error) {
	if index >= uint32(len(b.state.ValidatorRegistry)) {
		return nil, fmt.Errorf("Index out of bounds")
	}

	return &b.state.ValidatorRegistry[index], nil
}

// GetCommitteeValidatorIndices gets all validators in a committee at slot for shard with ID of shardID
func (b *Blockchain) GetCommitteeValidatorIndices(slot uint64, shardID uint64) ([]uint32, error) {
	return b.state.GetCommitteeIndices(slot, shardID, b.config)
}
