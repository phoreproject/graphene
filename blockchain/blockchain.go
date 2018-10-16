package blockchain

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/phoreproject/synapse/transaction"

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

const (
	// InitialForkVersion is version #1 of the chain.
	InitialForkVersion = iota
)

// InitializeState initializes state to the genesis state according to the config.
func (b Blockchain) InitializeState(initialValidators []InitialValidatorEntry) {
	b.state.Crystallized.Validators = make([]primitives.Validator, len(initialValidators))
	for _, v := range initialValidators {
		b.AddValidator(b.state.Crystallized.Validators, v.PubKey, v.ProofOfPossession, v.WithdrawalShard, v.WithdrawalAddress, v.RandaoCommitment, 0)
	}

	x := b.GetNewShuffling(zeroHash, 0)

	crosslinks := make([]primitives.Crosslink, b.config.ShardCount)

	for i := 0; i < b.config.ShardCount; i++ {
		crosslinks[i] = primitives.Crosslink{
			RecentlyChanged: false,
			Slot:            0,
			Hash:            &zeroHash,
		}
	}

	b.state.Crystallized = primitives.CrystallizedState{
		Validators:                  b.state.Crystallized.Validators,
		ValidatorSetChangeSlot:      0,
		Crosslinks:                  crosslinks,
		LastStateRecalculation:      0,
		LastFinalizedSlot:           0,
		LastJustifiedSlot:           0,
		JustifiedStreak:             0,
		ShardAndCommitteeForSlots:   append(x, x...),
		DepositsPenalizedInPeriod:   []uint32{},
		ValidatorSetDeltaHashChange: zeroHash,
		PreForkVersion:              InitialForkVersion,
		PostForkVersion:             InitialForkVersion,
		ForkSlotNumber:              0,
	}

	recentBlockHashes := make([]chainhash.Hash, b.config.CycleLength*2)
	for i := 0; i < b.config.CycleLength*2; i++ {
		recentBlockHashes[i] = zeroHash
	}

	b.state.Active = primitives.ActiveState{
		PendingActions:      []transaction.Transaction{},
		PendingAttestations: []transaction.Attestation{},
		RecentBlockHashes:   recentBlockHashes,
		RandaoMix:           zeroHash,
		Balances:            make(map[serialization.Address]uint64),
	}
}

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
func (b *Blockchain) GetActiveValidatorIndices() []uint32 {
	var l []uint32
	for i, v := range b.state.Crystallized.Validators {
		if v.Status == Active {
			l = append(l, uint32(i))
		}
	}
	return l
}

// ShuffleValidators shuffles an array of ints given a seed.
func ShuffleValidators(toShuffle []uint32, seed chainhash.Hash) []uint32 {
	shuffled := toShuffle[:]
	numValues := len(toShuffle)

	randBytes := 3
	randMax := uint32(math.Pow(2, float64(randBytes*8)) - 1)

	source := seed
	index := 0
	for index < numValues-1 {
		source = chainhash.HashH(source[:])
		for position := 0; position < (32 - (32 % randBytes)); position += randBytes {
			remaining := uint32(numValues - index)
			if remaining == 1 {
				break
			}

			sampleFromSource := binary.BigEndian.Uint32(append([]byte{'\x00'}, source[position:position+randBytes]...))

			sampleMax := randMax - randMax%remaining

			if sampleFromSource < sampleMax {
				replacementPos := (sampleFromSource % remaining) + uint32(index)
				shuffled[index], shuffled[replacementPos] = shuffled[replacementPos], shuffled[index]
				index++
			}
		}
	}
	return shuffled
}

// Split splits an array into N different sections.
func Split(l []uint32, splitCount uint32) [][]uint32 {
	out := make([][]uint32, splitCount)
	numItems := uint32(len(l))
	for i := uint32(0); i < splitCount; i++ {
		out[i] = l[(numItems * i / splitCount):(numItems * (i + 1) / splitCount)]
	}
	return out
}

// GetNewShuffling calculates the new shuffling of validators
// to slots and shards.
func (b *Blockchain) GetNewShuffling(seed chainhash.Hash, crosslinkingStart int) [][]primitives.ShardAndCommittee {
	activeValidators := b.GetActiveValidatorIndices()
	numActiveValidators := len(activeValidators)

	committeesPerSlot := numActiveValidators/b.config.CycleLength/(b.config.MinCommitteeSize*2) + 1
	// clamp between 1 and b.config.ShardCount / b.config.CycleLength
	if committeesPerSlot < 1 {
		committeesPerSlot = 1
	} else if committeesPerSlot > b.config.ShardCount/b.config.CycleLength {
		committeesPerSlot = b.config.ShardCount / b.config.CycleLength
	}

	output := make([][]primitives.ShardAndCommittee, b.config.CycleLength)

	shuffledValidatorIndices := ShuffleValidators(activeValidators, seed)

	validatorsPerSlot := Split(shuffledValidatorIndices, uint32(b.config.CycleLength))

	for slot, slotIndices := range validatorsPerSlot {
		shardIndices := Split(slotIndices, uint32(committeesPerSlot))

		shardIDStart := crosslinkingStart + slot*committeesPerSlot

		shardCommittees := make([]primitives.ShardAndCommittee, len(shardIndices))
		for shardPosition, indices := range shardIndices {
			shardCommittees[shardPosition] = primitives.ShardAndCommittee{
				ShardID:   uint32((shardIDStart + shardPosition) % b.config.ShardCount),
				Committee: indices,
			}
		}

		output[slot] = shardCommittees
	}
	return output
}

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
