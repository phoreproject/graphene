package blockchain

import (
	"errors"
	"math"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/serialization"
	"github.com/phoreproject/synapse/transaction"
)

// CrystallizedState is state that is updated every epoch
type CrystallizedState struct {
	LastStateRecalculation uint64
	JustifiedStreak        uint64
	LastJustifiedSlot      uint64
	LastFinalizedSlot      uint64
	CurrentDynasty         uint64
	DynastySeed            []byte
	DynastyStart           uint64
	Crosslinks             []Crosslink
	Validators             []Validator
	Committees             []ShardAndCommittee
}

// Validator is a single validator session (logging in and out)
type Validator struct {
	Pubkey            [32]byte
	WithdrawalAddress serialization.Address
	WithdrawalShardID uint64
	RandaoCommitment  *chainhash.Hash
	Balance           uint64
	StartDynasty      uint64
	EndDynasty        uint64
}

// Crosslink goes in a collation to represent the last crystallized beacon block.
type Crosslink struct {
	// Dynasty is the current dynasty number.
	Dynasty uint64

	// Slot is the slot within the current dynasty.
	Slot uint64

	// Hash is the hash of the beacon chain block.
	Hash *chainhash.Hash
}

// ShardAndCommittee keeps track of the validators assigned to a specific shard.
type ShardAndCommittee struct {
	// ShardID is which shard the committee is assigned to.
	ShardID uint64

	// Committee is the validator IDs that are assigned to this shard.
	Committee []uint32
}

// ActiveState is state that can change every block.
type ActiveState struct {
	PendingAttestations []transaction.Attestation
	PendingValidators   []Validator
	Balances            map[serialization.Address]uint64
}

// State is active and crystallized state.
type State struct {
	Active       ActiveState
	Crystallized CrystallizedState
}

// ValidateAttestation checks attestation invariants and the BLS signature.
func (b Blockchain) ValidateAttestation(s State, attestation transaction.Attestation, block BlockHeader, parentBlock BlockHeader, c Config) error {
	if attestation.Slot > parentBlock.SlotNumber {
		return errors.New("attestation slot number too high")
	}

	if !(attestation.Slot >= uint64(math.Max(float64(parentBlock.SlotNumber-uint64(c.CycleLength)+1), 0))) {
		return errors.New("attestation slot number too low")
	}

	if attestation.JustifiedSlot > s.Crystallized.LastJustifiedSlot {
		return errors.New("last justified slot should be less than or equal to the crystallized slot")
	}

	justifiedBlock, err := b.index.GetBlockNodeByHash(attestation.JustifiedBlockHash)
	if err != nil {
		return errors.New("justified block not in index")
	}

	if justifiedBlock.SlotNumber != attestation.Slot {
		return errors.New("justified slot does not match attestation")
	}

	// TODO: validate BLS sig

	return nil
}
