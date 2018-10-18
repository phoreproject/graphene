package blockchain

import (
	"errors"

	"github.com/phoreproject/synapse/db"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/serialization"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

var zeroHash = chainhash.Hash{}

// Blockchain represents a chain of blocks.
type Blockchain struct {
	chain     []chainhash.Hash
	db        db.Database
	config    *Config
	state     State
	voteCache map[chainhash.Hash]*VoteCache
}

// NewBlockchain creates a new blockchain.
func NewBlockchain(db db.Database, config *Config) Blockchain {
	return Blockchain{db: db, config: config}
}

// InitialValidatorEntry is the validator entry to be added
// at the beginning of a blockchain.
type InitialValidatorEntry struct {
	PubKey            []byte
	ProofOfPossession []byte
	WithdrawalShard   uint32
	WithdrawalAddress serialization.Address
	RandaoCommitment  chainhash.Hash
}

const (
	// InitialForkVersion is version #1 of the chain.
	InitialForkVersion = iota
)

// UpdateChainHead updates the blockchain head if needed
func (b *Blockchain) UpdateChainHead(n *primitives.Block) {
	if int64(n.SlotNumber) > int64(len(b.chain)-1) {
		b.SetTip(n)
	}
}

// SetTip sets the tip of the chain.
func (b *Blockchain) SetTip(n *primitives.Block) error {
	needed := n.SlotNumber + 1
	if uint64(cap(b.chain)) < needed {
		nodes := make([]chainhash.Hash, needed, needed+100)
		copy(nodes, b.chain)
		b.chain = nodes
	} else {
		prevLen := int32(len(b.chain))
		b.chain = b.chain[0:needed]
		for i := prevLen; uint64(i) < needed; i++ {
			b.chain[i] = zeroHash
		}
	}

	current := n

	for current != nil && b.chain[current.SlotNumber] != current.Hash() {
		b.chain[n.SlotNumber] = n.Hash()
		nextBlock, err := b.db.GetBlockForHash(n.AncestorHashes[0])
		if err != nil {
			return errors.New("block data is corrupted")
		}
		current = nextBlock
	}
	return nil
}

// Tip returns the block at the tip of the chain.
func (b Blockchain) Tip() chainhash.Hash {
	return b.chain[len(b.chain)-1]
}

// GetNodeByHeight gets a node from the active blockchain by height.
func (b Blockchain) GetNodeByHeight(height int64) chainhash.Hash {
	return b.chain[height]
}

// Height returns the height of the chain.
func (b Blockchain) Height() int {
	return len(b.chain) - 1
}
