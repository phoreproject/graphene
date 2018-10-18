package primitives

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/serialization"
)

// Validator is a single validator session (logging in and out)
type Validator struct {
	Pubkey            [32]byte
	WithdrawalAddress serialization.Address
	WithdrawalShardID uint32
	RandaoCommitment  chainhash.Hash
	RandaoLastChange  uint64
	Balance           uint64
	Status            uint8
	ExitSlot          uint64
}

// Crosslink goes in a collation to represent the last crystallized beacon block.
type Crosslink struct {
	RecentlyChanged bool

	// Slot is the slot within the current dynasty.
	Slot uint64

	// Hash is the hash of the beacon chain block.
	Hash *chainhash.Hash
}

// ShardAndCommittee keeps track of the validators assigned to a specific shard.
type ShardAndCommittee struct {
	// ShardID is which shard the committee is assigned to.
	ShardID uint32

	// Committee is the validator IDs that are assigned to this shard.
	Committee []uint32
}
