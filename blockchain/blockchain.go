package blockchain

import (
	"errors"

	"github.com/phoreproject/synapse/db"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/serialization"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

var zeroHash = chainhash.Hash{}

// BlockNode is a block header with a reference to the
// last block.
type BlockNode struct {
	primitives.BlockHeader
	Height   uint64
	PrevNode *BlockNode
}

// BlockIndex is an in-memory store of block headers.
type BlockIndex struct {
	index map[chainhash.Hash]*BlockNode
}

// NewBlockIndex creates and initializes a new block index.
func NewBlockIndex() BlockIndex {
	return BlockIndex{index: make(map[chainhash.Hash]*BlockNode)}
}

// GetBlockNodeByHash gets a block node by the given hash from the index.
func (b BlockIndex) GetBlockNodeByHash(h chainhash.Hash) (*BlockNode, error) {
	o, found := b.index[h]
	if !found {
		return nil, errors.New("could not find block in index")
	}
	return o, nil
}

// AddNode adds a node to the block index.
func (b BlockIndex) AddNode(node *BlockNode) {
	h := serialization.GetHash(&node.BlockHeader)
	b.index[h] = node
}

// Blockchain represents a chain of blocks.
type Blockchain struct {
	index  BlockIndex
	chain  []*BlockNode
	db     db.Database
	config Config
	state  primitives.State
}

// NewBlockchain creates a new blockchain.
func NewBlockchain(index BlockIndex, db db.Database, config Config) Blockchain {
	return Blockchain{index: index, db: db, config: config}
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

// NewBlockchainStartup creates a new blockchain from scratch.
// func NewBlockchainStartup(index BlockIndex, db db.Database, config Config, initialValidators []InitialValidatorEntry) *Blockchain {
// 	b := &Blockchain{index: index, db: db, config: config}
// 	validators := make([]primitives.Validator, len(initialValidators))
// 	for _, v := range initialValidators {
// 		b.AddValidator(validators, v.PubKey, v.ProofOfPossession, v.WithdrawalShard, v.WithdrawalAddress, v.RandaoCommitment, 0)
// 	}

// }

// MinEmptyValidator finds the first validator slot that is empty.
func MinEmptyValidator(validators []primitives.Validator) int {
	for i, v := range validators {
		if v.Status == Withdrawn {
			return i
		}
	}
	return -1
}

// AddValidator adds a validator to the current validator set.
func (b *Blockchain) AddValidator(currentValidators []primitives.Validator, pubkey []byte, proofOfPossession []byte, withdrawalShard uint32, withdrawalAddress serialization.Address, randaoCommitment chainhash.Hash, currentSlot uint64) (uint32, error) {
	verifies := true // blsig.Verify(chainhash.HashB(pubkey), pubkey, proofOfPossession)
	if !verifies {
		return 0, errors.New("validator proof of possesion does not verify")
	}

	var pk [32]byte
	copy(pk[:], pubkey)

	rec := primitives.Validator{
		Pubkey:            pk,
		WithdrawalAddress: withdrawalAddress,
		WithdrawalShardID: withdrawalShard,
		RandaoCommitment:  &randaoCommitment,
		Balance:           b.config.DepositSize,
		Status:            PendingActivation,
		ExitSlot:          0,
	}

	index := MinEmptyValidator(currentValidators)
	if index == -1 {
		currentValidators = append(currentValidators, rec)
		return uint32(len(currentValidators) - 1), nil
	}
	currentValidators[index] = rec
	return uint32(index), nil
}

// GetActiveValidatorIndices gets the indices of active validators.
func (b *Blockchain) GetActiveValidatorIndices() []int {
	var l []int
	for i, v := range b.state.Crystallized.Validators {
		if v.Status == Active {
			l = append(l, i)
		}
	}
	return l
}

// func ShuffleValidators(toShuffle []int, seed chainhash.Hash) {
// 	h := chainhash.HashH(seed[:])

// }

// UpdateChainHead updates the blockchain head if needed
func (b *Blockchain) UpdateChainHead(n *BlockNode) {
	if int64(n.Height) > int64(len(b.chain)-1) {
		b.SetTip(n)
	}
}

// SetTip sets the tip of the chain.
func (b *Blockchain) SetTip(n *BlockNode) {
	needed := n.Height + 1
	if uint64(cap(b.chain)) < needed {
		nodes := make([]*BlockNode, needed, needed+100)
		copy(nodes, b.chain)
		b.chain = nodes
	} else {
		prevLen := int32(len(b.chain))
		b.chain = b.chain[0:needed]
		for i := prevLen; uint64(i) < needed; i++ {
			b.chain[i] = nil
		}
	}

	for n != nil && b.chain[n.Height] != n {
		b.chain[n.Height] = n
		n = n.PrevNode
	}
}

// Tip returns the block at the tip of the chain.
func (b Blockchain) Tip() *BlockNode {
	return b.chain[len(b.chain)-1]
}

// GetNodeByHeight gets a node from the active blockchain by height.
func (b Blockchain) GetNodeByHeight(height int64) *BlockNode {
	return b.chain[height]
}

// Height returns the height of the chain.
func (b Blockchain) Height() int {
	return len(b.chain) - 1
}
