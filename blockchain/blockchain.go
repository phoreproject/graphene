package blockchain

import (
	"fmt"
	"sync"

	logger "github.com/inconshreveable/log15"
	"github.com/phoreproject/synapse/bls"

	"github.com/phoreproject/synapse/db"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/serialization"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

var zeroHash = chainhash.Hash{}

type blockchainView struct {
	chain []chainhash.Hash
	lock  *sync.Mutex
}

func (bv *blockchainView) GetBlock(n int) (*chainhash.Hash, error) {
	bv.lock.Lock()
	defer bv.lock.Unlock()
	if len(bv.chain) > n && n >= 0 {
		return &bv.chain[n], nil
	}
	return nil, fmt.Errorf("block %d does not exist", n)
}

func (bv *blockchainView) Height() int {
	bv.lock.Lock()
	defer bv.lock.Unlock()
	fmt.Println(len(bv.chain))
	return len(bv.chain) - 1
}

// Blockchain represents a chain of blocks.
type Blockchain struct {
	chain     blockchainView
	db        db.Database
	config    *Config
	state     State
	stateLock *sync.Mutex
	voteCache map[chainhash.Hash]*VoteCache
}

// NewBlockchainWithInitialValidators creates a new blockchain with the specified
// initial validators.
func NewBlockchainWithInitialValidators(db db.Database, config *Config, validators []InitialValidatorEntry) (*Blockchain, error) {
	b := &Blockchain{
		db:     db,
		config: config,
		chain: blockchainView{
			chain: []chainhash.Hash{},
			lock:  &sync.Mutex{},
		},
		stateLock: &sync.Mutex{},
	}
	err := b.InitializeState(validators)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// InitialValidatorEntry is the validator entry to be added
// at the beginning of a blockchain.
type InitialValidatorEntry struct {
	PubKey            bls.PublicKey
	ProofOfPossession bls.Signature
	WithdrawalShard   uint32
	WithdrawalAddress serialization.Address
	RandaoCommitment  chainhash.Hash
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
	if int64(n.SlotNumber) > int64(len(b.chain.chain)-1) {
		logger.Debug("updating blockchain head", "hash", n.Hash(), "height", n.SlotNumber)
		b.chain.lock.Unlock()
		err := b.SetTip(n)
		if err != nil {
			return err
		}
	} else {
		b.chain.lock.Unlock()
	}
	return nil
}

// SetTip sets the tip of the chain.
func (b *Blockchain) SetTip(n *primitives.Block) error {
	b.chain.lock.Lock()
	defer b.chain.lock.Unlock()
	needed := n.SlotNumber + 1
	if uint64(cap(b.chain.chain)) < needed {
		nodes := make([]chainhash.Hash, needed, needed+100)
		copy(nodes, b.chain.chain)
		b.chain.chain = nodes
	} else {
		prevLen := int32(len(b.chain.chain))
		b.chain.chain = b.chain.chain[0:needed]
		for i := prevLen; uint64(i) < needed; i++ {
			b.chain.chain[i] = zeroHash
		}
	}

	current := n

	for current != nil && b.chain.chain[current.SlotNumber] != current.Hash() {
		b.chain.chain[n.SlotNumber] = n.Hash()
		nextBlock, err := b.db.GetBlockForHash(n.AncestorHashes[0])
		if err != nil {
			nextBlock = nil
		}
		current = nextBlock
	}

	return nil
}

// Tip returns the block at the tip of the chain.
func (b Blockchain) Tip() chainhash.Hash {
	b.chain.lock.Lock()
	tip := b.chain.chain[len(b.chain.chain)-1]
	b.chain.lock.Unlock()
	return tip
}

// GetNodeByHeight gets a node from the active blockchain by height.
func (b Blockchain) GetNodeByHeight(height uint64) (chainhash.Hash, error) {
	b.chain.lock.Lock()
	defer b.chain.lock.Unlock()
	if uint64(len(b.chain.chain)-1) < height {
		return chainhash.Hash{}, fmt.Errorf("attempted to retrieve block hash of height > chain height")
	}
	node := b.chain.chain[height]
	return node, nil
}

// Height returns the height of the chain.
func (b *Blockchain) Height() int {
	return b.chain.Height()
}

// LastBlock gets the last block in the chain
func (b *Blockchain) LastBlock() (*primitives.Block, error) {
	hash, err := b.chain.GetBlock(b.chain.Height())
	if err != nil {
		return nil, err
	}
	block, err := b.db.GetBlockForHash(*hash)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GetConfig returns the config used by this blockchain
func (b *Blockchain) GetConfig() *Config {
	return b.config
}

// GetSlotAndShardAssignment gets the shard and slot assignment for a specific
// validator.
func (b *Blockchain) GetSlotAndShardAssignment(validatorID uint32) (uint32, uint64, int, error) {
	earliestSlotInArray := int(b.state.Crystallized.LastStateRecalculation) - b.config.CycleLength
	if earliestSlotInArray < 0 {
		earliestSlotInArray = 0
	}
	for i, slot := range b.state.Crystallized.ShardAndCommitteeForSlots {
		for j, committee := range slot {
			for v, validator := range committee.Committee {
				if uint32(validator) != validatorID {
					continue
				}
				if j == 0 && v == i%len(committee.Committee) {
					return committee.ShardID, uint64(i), RoleProposer, nil
				}
				return committee.ShardID, uint64(i), RoleAttester, nil
			}
		}
	}
	return 0, 0, 0, fmt.Errorf("validator not found in set %d", validatorID)
}

func (b *Blockchain) GetValidatorAtIndex(index uint32) (*primitives.Validator, error) {
	if index >= uint32(len(b.state.Crystallized.Validators)) {
		return nil, fmt.Errorf("Index out of bounds")
	}

	return &b.state.Crystallized.Validators[index], nil
}
