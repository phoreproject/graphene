package primitives

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/serialization"
)

// Validator is a single validator session (logging in and out)
type Validator struct {
	// BLS public key
	Pubkey bls.PublicKey
	// Withdrawal credentials
	WithdrawalCredentials serialization.Address
	// RANDAO commitment
	RandaoCommitment chainhash.Hash
	// Slot the RANDAO commitment was last changed
	RandaoLastChange uint64
	Balance          uint64
	// Status code
	Status uint8
	// Slot when validator last changed status (or 0)
	LastStatusChangeSlot uint64
	// Sequence number when validator exited (or 0)
	ExitSeq uint64
}

// ToProto creates a ProtoBuf ValidatorResponse from a Validator
func (v *Validator) ToProto() *pb.ValidatorResponse {
	return &pb.ValidatorResponse{
		Pubkey:                0, //v.Pubkey,
		WithdrawalCredentials: v.WithdrawalCredentials[:],
		RandaoCommitment:      v.RandaoCommitment[:],
		RandaoLastChange:      v.RandaoLastChange,
		Balance:               v.Balance,
		Status:                uint32(v.Status),
		ExitSeq:               v.ExitSeq}
}

// Crosslink goes in a collation to represent the last crystallized beacon block.
type Crosslink struct {
	// Slot is the slot within the current dynasty.
	Slot uint64

	// Shard chain block hash
	ShardBlockHash chainhash.Hash
}

// ShardAndCommittee keeps track of the validators assigned to a specific shard.
type ShardAndCommittee struct {
	// Shard number
	Shard uint64

	// Validator indices
	Committee []uint32
}
