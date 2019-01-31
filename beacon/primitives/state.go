package primitives

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
)

// ForkData represents the fork information
type ForkData struct {
	// Previous fork version
	PreForkVersion uint64
	// Post fork version
	PostForkVersion uint64
	// Fork slot number
	ForkSlotNumber uint64
}

// Copy returns a copy of the fork data.
func (f ForkData) Copy() ForkData {
	return ForkData{PreForkVersion: f.PreForkVersion, PostForkVersion: f.PostForkVersion, ForkSlotNumber: f.ForkSlotNumber}
}

// State is the state of a beacon block
type State struct {
	// MISC ITEMS
	// Slot is the current slot.
	Slot uint64

	// GenesisTime is the time of the genesis block.
	GenesisTime uint64

	// ForkData is the versioning data for hard forks.
	ForkData ForkData

	// VALIDATOR REGISTRY
	// ValidatorRegistry is the registry mapping IDs to validators
	ValidatorRegistry []Validator

	// ValidatorBalances are the balances corresponding to each validator.
	ValidatorBalances []uint64

	// ValidatorRegistryLatestChangeSlot is the slot where the validator registry
	// last changed.
	ValidatorRegistryLatestChangeSlot uint64

	// ValidatorRegistryExitCount is the number of validators that are exited.
	ValidatorRegistryExitCount uint64

	// ValidatorSetDeltaHashChange is for light clients to keep track of validator
	// registry changes.
	ValidatorRegistryDeltaChainTip chainhash.Hash

	// RANDOMNESS
	// RandaoMix is the mix of randao reveals to provide entropy.
	RandaoMix chainhash.Hash

	// NextSeed is the next RANDAO seed.
	NextSeed chainhash.Hash

	// COMMITTEES
	// ShardAndCommitteeForSlots is a list of committee members
	// and their assigned shard, per slot
	ShardAndCommitteeForSlots [][]ShardAndCommittee

	// FINALITY
	PreviousJustifiedSlot uint64
	JustifiedSlot         uint64
	JustificationBitfield uint64
	FinalizedSlot         uint64

	// RECENT STATE
	LatestCrosslinks            []Crosslink
	LatestBlockHashes           []chainhash.Hash
	LatestPenalizedExitBalances []uint64
	LatestAttestations          []PendingAttestation
	BatchedBlockRoots           []chainhash.Hash
}

// Copy deep-copies the state.
func (s *State) Copy() State {
	newValidatorRegistry := make([]Validator, len(s.ValidatorRegistry))
	for i := range s.ValidatorRegistry {
		newValidatorRegistry[i] = s.ValidatorRegistry[i].Copy()
	}
	newValidatorBalances := make([]uint64, len(s.ValidatorBalances))
	for i := range s.ValidatorBalances {
		newValidatorBalances[i] = s.ValidatorBalances[i]
	}
	var newValidatorRegistryDeltaChainTip chainhash.Hash
	var newRandaoMix chainhash.Hash
	var newNextSeed chainhash.Hash
	copy(newValidatorRegistryDeltaChainTip[:], s.ValidatorRegistryDeltaChainTip[:])
	copy(newRandaoMix[:], s.RandaoMix[:])
	copy(newNextSeed[:], s.NextSeed[:])

	newShardAndCommitteeForSlots := make([][]ShardAndCommittee, len(s.ShardAndCommitteeForSlots))
	for i, slot := range s.ShardAndCommitteeForSlots {
		newShardAndCommitteeForSlots[i] = make([]ShardAndCommittee, len(slot))
		for n, committee := range slot {
			newShardAndCommitteeForSlots[i][n] = committee.Copy()
		}
	}

	newLatestCrosslinks := make([]Crosslink, len(s.LatestCrosslinks))
	copy(newLatestCrosslinks, s.LatestCrosslinks)
	newLatestBlockHashes := make([]chainhash.Hash, len(s.LatestBlockHashes))
	copy(newLatestBlockHashes, s.LatestBlockHashes)
	newLatestPenalizedExitBalances := make([]uint64, len(s.LatestPenalizedExitBalances))
	copy(newLatestPenalizedExitBalances, s.LatestPenalizedExitBalances)
	newLatestAttestations := make([]PendingAttestation, len(s.LatestAttestations))
	for i := range s.LatestAttestations {
		newLatestAttestations[i] = s.LatestAttestations[i].Copy()
	}
	newBatchedBlockRoots := make([]chainhash.Hash, len(s.BatchedBlockRoots))
	copy(newBatchedBlockRoots, s.BatchedBlockRoots)
	newState := State{
		Slot:                              s.Slot,
		GenesisTime:                       s.GenesisTime,
		ForkData:                          s.ForkData.Copy(),
		ValidatorRegistry:                 newValidatorRegistry,
		ValidatorBalances:                 newValidatorBalances,
		ValidatorRegistryLatestChangeSlot: s.ValidatorRegistryLatestChangeSlot,
		ValidatorRegistryExitCount:        s.ValidatorRegistryExitCount,
		ValidatorRegistryDeltaChainTip:    newValidatorRegistryDeltaChainTip,
		RandaoMix:                         newRandaoMix,
		NextSeed:                          newNextSeed,
		ShardAndCommitteeForSlots:         newShardAndCommitteeForSlots,
		PreviousJustifiedSlot:             s.PreviousJustifiedSlot,
		JustifiedSlot:                     s.JustifiedSlot,
		JustificationBitfield:             s.JustificationBitfield,
		FinalizedSlot:                     s.FinalizedSlot,
		LatestCrosslinks:                  newLatestCrosslinks,
		LatestBlockHashes:                 newLatestBlockHashes,
		LatestPenalizedExitBalances:       newLatestPenalizedExitBalances,
		LatestAttestations:                newLatestAttestations,
		BatchedBlockRoots:                 newBatchedBlockRoots,
	}

	return newState
}

// GetEffectiveBalance gets the effective balance for a validator
func (s *State) GetEffectiveBalance(index uint32, c *config.Config) uint64 {
	if s.ValidatorBalances[index] <= c.MaxDeposit {
		return s.ValidatorBalances[index]
	}
	return c.MaxDeposit
}

// GetNewValidatorRegistryDeltaChainTip gets the new delta chain tip hash.
func GetNewValidatorRegistryDeltaChainTip(currentValidatorRegistryDeltaChainTip chainhash.Hash, validatorIndex uint32, pubkey bls.PublicKey, flag uint64) chainhash.Hash {
	// TODO: fix me!
	p := &pb.ValidatorRegistryDeltaBlock{
		LatestRegistryDeltaRoot: currentValidatorRegistryDeltaChainTip[:],
		ValidatorIndex:          validatorIndex,
		Pubkey:                  pubkey.Serialize(),
		Flag:                    flag,
	}
	pBytes, _ := proto.Marshal(p)
	return chainhash.HashH(pBytes)
}

const (
	// ActivationFlag is a flag to indicate a validator is joining
	ActivationFlag = iota
	// ExitFlag is a flag to indeicate a validator is leaving
	ExitFlag
)

// ActivateValidator activates a validator in the state at a certain index.
func (s *State) ActivateValidator(index uint32) error {
	validator := s.ValidatorRegistry[index]
	if validator.Status != PendingActivation {
		return errors.New("validator is not pending activation")
	}

	validator.Status = Active
	validator.LatestStatusChangeSlot = s.Slot
	s.ValidatorRegistryDeltaChainTip = GetNewValidatorRegistryDeltaChainTip(s.ValidatorRegistryDeltaChainTip, index, validator.Pubkey, ActivationFlag)
	return nil
}

// InitiateValidatorExit moves a validator from active to pending exit.
func (s *State) InitiateValidatorExit(index uint32) error {
	validator := s.ValidatorRegistry[index]
	if validator.Status != Active {
		return errors.New("validator is not active")
	}

	validator.Status = ActivePendingExit
	validator.LatestStatusChangeSlot = s.Slot
	return nil
}

// GetBeaconProposerIndex gets the validator index of the block proposer at a certain
// slot.
func (s *State) GetBeaconProposerIndex(slot uint64, c *config.Config) uint32 {
	firstCommittee := s.GetShardCommitteesAtSlot(slot, c)[0].Committee
	return firstCommittee[int(slot)%len(firstCommittee)]
}

// ExitValidator handles state changes when a validator exits.
func (s *State) ExitValidator(index uint32, status uint64, c *config.Config) {
	validator := s.ValidatorRegistry[index]
	prevStatus := validator.Status

	if prevStatus == ExitedWithPenalty {
		return
	}

	validator.Status = status
	validator.LatestStatusChangeSlot = s.Slot

	if status == ExitedWithPenalty {
		s.LatestPenalizedExitBalances[s.Slot/c.CollectivePenaltyCalculationPeriod] += s.GetEffectiveBalance(index, c)

		whistleblowerIndex := s.GetBeaconProposerIndex(s.Slot, c)
		whistleblowerReward := s.GetEffectiveBalance(index, c) / c.WhistleblowerRewardQuotient
		s.ValidatorBalances[whistleblowerIndex] += whistleblowerReward
		s.ValidatorBalances[index] -= whistleblowerReward
	}

	if prevStatus == ExitedWithoutPenalty {
		return
	}

	s.ValidatorRegistryExitCount++
	validator.ExitCount = s.ValidatorRegistryExitCount
	s.ValidatorRegistryDeltaChainTip = GetNewValidatorRegistryDeltaChainTip(s.ValidatorRegistryDeltaChainTip, index, validator.Pubkey, ExitFlag)
}

// UpdateValidatorStatus moves a validator to a specific status.
func (s *State) UpdateValidatorStatus(index uint32, status uint64, c *config.Config) error {
	if status == Active {
		err := s.ActivateValidator(index)
		return err
	} else if status == ActivePendingExit {
		err := s.InitiateValidatorExit(index)
		return err
	} else if status == ExitedWithPenalty || status == ExitedWithoutPenalty {
		s.ExitValidator(index, status, c)
	}
	return nil
}

// ShardCommitteeByShardID gets the shards committee from a list of committees/shards
// in a list.
func ShardCommitteeByShardID(shardID uint64, shardCommittees []ShardAndCommittee) ([]uint32, error) {
	for _, s := range shardCommittees {
		if s.Shard == shardID {
			return s.Committee, nil
		}
	}

	return nil, fmt.Errorf("unable to find committee based on shard: %v", shardID)
}

// CommitteeInShardAndSlot gets the committee of validator indices at a specific
// shard and slot given the relative slot number [0, CYCLE_LENGTH] and shard ID.
func CommitteeInShardAndSlot(slotIndex uint64, shardID uint64, shardCommittees [][]ShardAndCommittee) ([]uint32, error) {
	shardCommittee := shardCommittees[slotIndex]

	return ShardCommitteeByShardID(shardID, shardCommittee)
}

// GetShardCommitteesAtSlot gets the committees assigned to a specific slot.
func (s *State) GetShardCommitteesAtSlot(slot uint64, c *config.Config) []ShardAndCommittee {
	earliestSlot := s.Slot - (s.Slot % c.EpochLength) - c.EpochLength
	return s.ShardAndCommitteeForSlots[s.Slot-earliestSlot]
}

// GetAttesterIndices gets all of the validator indices involved with the committee
// assigned to the shard and slot of the committee.
func (s *State) GetAttesterIndices(slot uint64, shard uint64, con *config.Config) ([]uint32, error) {
	slotsStart := (s.Slot % con.EpochLength) - con.EpochLength
	slotIndex := (slot - slotsStart) % uint64(con.EpochLength)
	return CommitteeInShardAndSlot(slotIndex, shard, s.ShardAndCommitteeForSlots)
}

// GetAttesterCommitteeSize gets the size of committee
func (s *State) GetAttesterCommitteeSize(slot uint64, con *config.Config) uint32 {
	slotsStart := (s.Slot % con.EpochLength) - con.EpochLength
	slotIndex := (slot - slotsStart) % uint64(con.EpochLength)
	return uint32(len(s.ShardAndCommitteeForSlots[slotIndex]))
}

// GetCommitteeIndices gets all of the validator indices involved with the committee
// assigned to the shard and slot of the committee.
func (s *State) GetCommitteeIndices(slot uint64, shardID uint64, con *config.Config) ([]uint32, error) {
	slotsStart := (s.Slot % con.EpochLength) - con.EpochLength
	slotIndex := (slot - slotsStart) % uint64(con.EpochLength)
	return CommitteeInShardAndSlot(slotIndex, shardID, s.ShardAndCommitteeForSlots)
}

// ValidateProofOfPossession validates a proof of possession for a new validator.
func (s *State) ValidateProofOfPossession(pubkey bls.PublicKey, proofOfPossession bls.Signature, withdrawalCredentials chainhash.Hash, randaoCommitment chainhash.Hash, pocCommitment chainhash.Hash) (bool, error) {
	proofOfPossessionData := &pb.DepositParameters{
		PublicKey:             pubkey.Serialize(),
		WithdrawalCredentials: withdrawalCredentials[:],
		RandaoCommitment:      randaoCommitment[:],
		PoCCommitment:         pocCommitment[:],
		ProofOfPossession:     bls.EmptySignature.Serialize(),
	}

	proofOfPossessionDataBytes, err := proto.Marshal(proofOfPossessionData)
	if err != nil {
		return false, err
	}

	valid, err := bls.VerifySig(&pubkey, proofOfPossessionDataBytes, &proofOfPossession)
	if err != nil {
		return false, err
	}
	return valid, nil
}

// MinEmptyValidator finds the first validator slot that is empty.
func MinEmptyValidator(validators []Validator, validatorBalances []uint64, c *config.Config, currentSlot uint64) int {
	for i := range validators {
		if validatorBalances[i] == 0 && validators[i].LatestStatusChangeSlot+c.ZeroBalanceValidatorTTL <= currentSlot {
			return i
		}
	}
	return -1
}

// ProcessDeposit processes a deposit with the context of the current state.
func (s *State) ProcessDeposit(pubkey bls.PublicKey, amount uint64, proofOfPossession bls.Signature, withdrawalCredentials chainhash.Hash, randaoCommitment chainhash.Hash, pocCommitment chainhash.Hash, c *config.Config) (uint32, error) {
	sigValid, err := s.ValidateProofOfPossession(pubkey, proofOfPossession, withdrawalCredentials, randaoCommitment, pocCommitment)
	if err != nil {
		return 0, err
	}
	if !sigValid {
		return 0, errors.New("invalid deposit signature")
	}

	validatorAlreadyRegisteredIndex := -1

	validatorPubkeys := make([]bls.PublicKey, len(s.ValidatorRegistry))
	for i := range s.ValidatorRegistry {
		validatorPubkeys[i] = s.ValidatorRegistry[i].Pubkey

		if s.ValidatorRegistry[i].Pubkey.Equals(pubkey) {
			validatorAlreadyRegisteredIndex = i
		}
	}

	var index int

	if validatorAlreadyRegisteredIndex == -1 {
		validator := Validator{
			Pubkey:                  pubkey,
			WithdrawalCredentials:   withdrawalCredentials,
			RandaoCommitment:        randaoCommitment,
			RandaoSkips:             0,
			Status:                  PendingActivation,
			LatestStatusChangeSlot:  s.Slot,
			ExitCount:               0,
			PoCCommitment:           pocCommitment,
			LastPoCChangeSlot:       0,
			SecondLastPoCChangeSlot: 0,
		}

		index = MinEmptyValidator(s.ValidatorRegistry, s.ValidatorBalances, c, s.Slot)
		if index == -1 {
			s.ValidatorRegistry = append(s.ValidatorRegistry, validator)
			s.ValidatorBalances = append(s.ValidatorBalances, amount)
			index = len(s.ValidatorRegistry) - 1
		} else {
			s.ValidatorRegistry[index] = validator
			s.ValidatorBalances[index] = amount
		}
	} else {
		index = validatorAlreadyRegisteredIndex
		if !bytes.Equal(s.ValidatorRegistry[index].WithdrawalCredentials[:], withdrawalCredentials[:]) {
			return 0, errors.New("withdrawal credentials do not match")
		}

		s.ValidatorBalances[index] += amount
	}
	return uint32(index), nil
}

// TreeHashSSZ calculates the state hash for a certain state
func (s *State) TreeHashSSZ() (chainhash.Hash, error) {
	// TODO: fix me
	var slotBytes [8]byte
	binary.BigEndian.PutUint64(slotBytes[:], s.Slot)
	return chainhash.HashH(slotBytes[:]), nil
}

// ShardReassignmentRecord is the record of shard reassignment
type ShardReassignmentRecord struct {
	// Which validator to reassign
	ValidatorIndex uint32
	// To which shard
	Shard uint64
	// When
	Slot uint64
}

const (
	// Active is a status for a validator that is active.
	Active = iota
	// ActivePendingExit is a status for a validator that is active but pending exit.
	ActivePendingExit
	// PendingActivation is a status for a newly added validator
	PendingActivation
	// ExitedWithoutPenalty is a validator that gracefully exited
	ExitedWithoutPenalty
	// ExitedWithPenalty is a validator that exited not-so-gracefully
	ExitedWithPenalty
)

// Validator is a single validator session (logging in and out)
type Validator struct {
	// BLS public key
	Pubkey bls.PublicKey
	// Withdrawal credentials
	WithdrawalCredentials chainhash.Hash
	// RANDAO commitment
	RandaoCommitment chainhash.Hash
	// RandaoSkips is the slots the proposer has skipped.
	RandaoSkips uint64
	// Balance in satoshi.
	Balance uint64
	// Status code
	Status uint64
	// Slot when validator last changed status (or 0)
	LatestStatusChangeSlot uint64
	// Sequence number when validator exited (or 0)
	ExitCount uint64
	// PoCCommitment is the proof-of-custody commitment hash
	PoCCommitment chainhash.Hash
	// LastPoCChangeSlot is the last time the PoC was changed
	LastPoCChangeSlot uint64
	// SecondLastPoCChangeSlot is the second to last time the PoC was changed
	SecondLastPoCChangeSlot uint64
}

// Copy copies a validator instance.
func (v *Validator) Copy() Validator {
	newValidator := *v
	newValidator.Pubkey = v.Pubkey.Copy()
	return newValidator
}

// IsActive checks if the validator is active.
func (v Validator) IsActive() bool {
	return v.Status == Active || v.Status == ActivePendingExit
}

// GetActiveValidatorIndices gets validator indices that are active.
func GetActiveValidatorIndices(validators []Validator) []int {
	active := make([]int, 0, len(validators))
	for i, v := range validators {
		if v.IsActive() {
			active = append(active, i)
		}
	}
	return active
}

// ToProto creates a ProtoBuf ValidatorResponse from a Validator
func (v *Validator) ToProto() *pb.Validator {
	return &pb.Validator{
		Pubkey:                  v.Pubkey.Serialize(),
		WithdrawalCredentials:   v.WithdrawalCredentials[:],
		RandaoCommitment:        v.RandaoCommitment[:],
		PoCCommitment:           v.PoCCommitment[:],
		LastPoCChangeSlot:       v.LastPoCChangeSlot,
		SecondLastPoCChangeSlot: v.SecondLastPoCChangeSlot,
		RandaoSkips:             v.RandaoSkips,
		Status:                  v.Status,
		LatestStatusChangeSlot:  v.LatestStatusChangeSlot,
		ExitCount:               v.ExitCount}
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

	// Total validator count (for proofs of custody)
	TotalValidatorCount uint64
}

// Copy copies the ShardAndCommittee
func (sc *ShardAndCommittee) Copy() ShardAndCommittee {
	newSc := *sc
	copy(newSc.Committee, sc.Committee)
	return newSc
}
