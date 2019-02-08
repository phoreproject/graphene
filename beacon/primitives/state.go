package primitives

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/golang/protobuf/proto"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"

	"github.com/prysmaticlabs/prysm/shared/ssz"
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

// GetVersionForSlot returns the version for a specific slot number.
func (f *ForkData) GetVersionForSlot(slot uint64) uint64 {
	if slot < f.ForkSlotNumber {
		return f.PreForkVersion
	}
	return f.PostForkVersion
}

// EncodeSSZ implements Encodable
func (f ForkData) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, f.PreForkVersion); err != nil {
		return err
	}
	if err := ssz.Encode(writer, f.PostForkVersion); err != nil {
		return err
	}
	if err := ssz.Encode(writer, f.ForkSlotNumber); err != nil {
		return err
	}

	return nil
}

// EncodeSSZSize implements Encodable
func (f ForkData) EncodeSSZSize() (uint32, error) {
	var sizeOfPreForkVersion, sizeOfPostForkVersion, sizeOfForkSlotNumber uint32
	var err error
	if sizeOfPreForkVersion, err = ssz.EncodeSize(f.PreForkVersion); err != nil {
		return 0, err
	}
	if sizeOfPostForkVersion, err = ssz.EncodeSize(f.PostForkVersion); err != nil {
		return 0, err
	}
	if sizeOfForkSlotNumber, err = ssz.EncodeSize(f.ForkSlotNumber); err != nil {
		return 0, err
	}
	return sizeOfPreForkVersion + sizeOfPostForkVersion + sizeOfForkSlotNumber, nil
}

// DecodeSSZ implements Decodable
func (f ForkData) DecodeSSZ(reader io.Reader) error {
	if err := ssz.Decode(reader, &f.PreForkVersion); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &f.PostForkVersion); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &f.ForkSlotNumber); err != nil {
		return err
	}

	return nil
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

// EncodeSSZ implements Encodable
func (s State) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, s.Slot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.GenesisTime); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.ForkData); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.ValidatorRegistry); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.ValidatorBalances); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.ValidatorRegistryLatestChangeSlot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.ValidatorRegistryExitCount); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.ValidatorRegistryDeltaChainTip); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.RandaoMix); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.NextSeed); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.ShardAndCommitteeForSlots); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.PreviousJustifiedSlot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.JustifiedSlot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.JustificationBitfield); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.FinalizedSlot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.LatestCrosslinks); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.LatestBlockHashes); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.LatestPenalizedExitBalances); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.LatestAttestations); err != nil {
		return err
	}
	if err := ssz.Encode(writer, s.BatchedBlockRoots); err != nil {
		return err
	}
	return nil
}

// EncodeSSZSize implements Encodable
func (s State) EncodeSSZSize() (uint32, error) {
	var sizeOfslot, sizeOfgenesisTime, sizeOfforkData, sizeOfvalidatorRegistry,
		sizeOfvalidatorBalances, sizeOfvalidatorRegistryLatestChangeSlot, sizeOfvalidatorRegistryExitCount,
		sizeOfvalidatorRegistryDeltaChainTip, sizeOfrandaoMix, sizeOfnextSeed, sizeOfshardAndCommitteeForSlots,
		sizeOfpreviousJustifiedSlot, sizeOfjustifiedSlot, sizeOfjustificationBitfield, sizeOffinalizedSlot,
		sizeOflatestCrosslinks, sizeOflatestBlockHashes, sizeOflatestPenalizedExitBalances,
		sizeOflatestAttestations, sizeOfbatchedBlockRoots uint32
	var err error
	if sizeOfslot, err = ssz.EncodeSize(s.Slot); err != nil {
		return 0, err
	}
	if sizeOfgenesisTime, err = ssz.EncodeSize(s.GenesisTime); err != nil {
		return 0, err
	}
	if sizeOfforkData, err = ssz.EncodeSize(s.ForkData); err != nil {
		return 0, err
	}
	if sizeOfvalidatorRegistry, err = ssz.EncodeSize(s.ValidatorRegistry); err != nil {
		return 0, err
	}
	if sizeOfvalidatorBalances, err = ssz.EncodeSize(s.ValidatorBalances); err != nil {
		return 0, err
	}
	if sizeOfvalidatorRegistryLatestChangeSlot, err = ssz.EncodeSize(s.ValidatorRegistryLatestChangeSlot); err != nil {
		return 0, err
	}
	if sizeOfvalidatorRegistryExitCount, err = ssz.EncodeSize(s.ValidatorRegistryExitCount); err != nil {
		return 0, err
	}
	if sizeOfvalidatorRegistryDeltaChainTip, err = ssz.EncodeSize(s.ValidatorRegistryDeltaChainTip); err != nil {
		return 0, err
	}
	if sizeOfrandaoMix, err = ssz.EncodeSize(s.RandaoMix); err != nil {
		return 0, err
	}
	if sizeOfnextSeed, err = ssz.EncodeSize(s.NextSeed); err != nil {
		return 0, err
	}
	if sizeOfshardAndCommitteeForSlots, err = ssz.EncodeSize(s.ShardAndCommitteeForSlots); err != nil {
		return 0, err
	}
	if sizeOfpreviousJustifiedSlot, err = ssz.EncodeSize(s.PreviousJustifiedSlot); err != nil {
		return 0, err
	}
	if sizeOfjustifiedSlot, err = ssz.EncodeSize(s.JustifiedSlot); err != nil {
		return 0, err
	}
	if sizeOfjustificationBitfield, err = ssz.EncodeSize(s.JustificationBitfield); err != nil {
		return 0, err
	}
	if sizeOffinalizedSlot, err = ssz.EncodeSize(s.FinalizedSlot); err != nil {
		return 0, err
	}
	if sizeOflatestCrosslinks, err = ssz.EncodeSize(s.LatestCrosslinks); err != nil {
		return 0, err
	}
	if sizeOflatestBlockHashes, err = ssz.EncodeSize(s.LatestBlockHashes); err != nil {
		return 0, err
	}
	if sizeOflatestPenalizedExitBalances, err = ssz.EncodeSize(s.LatestPenalizedExitBalances); err != nil {
		return 0, err
	}
	if sizeOflatestAttestations, err = ssz.EncodeSize(s.LatestAttestations); err != nil {
		return 0, err
	}
	if sizeOfbatchedBlockRoots, err = ssz.EncodeSize(s.BatchedBlockRoots); err != nil {
		return 0, err
	}
	return sizeOfslot + sizeOfgenesisTime + sizeOfforkData + sizeOfvalidatorRegistry +
		sizeOfvalidatorBalances + sizeOfvalidatorRegistryLatestChangeSlot + sizeOfvalidatorRegistryExitCount +
		sizeOfvalidatorRegistryDeltaChainTip + sizeOfrandaoMix + sizeOfnextSeed + sizeOfshardAndCommitteeForSlots +
		sizeOfpreviousJustifiedSlot + sizeOfjustifiedSlot + sizeOfjustificationBitfield + sizeOffinalizedSlot +
		sizeOflatestCrosslinks + sizeOflatestBlockHashes + sizeOflatestPenalizedExitBalances +
		sizeOflatestAttestations + sizeOfbatchedBlockRoots, nil
}

// DecodeSSZ implements Decodable
func (s State) DecodeSSZ(reader io.Reader) error {
	if err := ssz.Decode(reader, &s.Slot); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &s.GenesisTime); err != nil {
		return err
	}
	s.ForkData = ForkData{}
	if err := ssz.Decode(reader, &s.ForkData); err != nil {
		return err
	}
	s.ValidatorRegistry = []Validator{}
	if err := ssz.Decode(reader, &s.ValidatorRegistry); err != nil {
		return err
	}
	s.ValidatorBalances = []uint64{}
	if err := ssz.Decode(reader, &s.ValidatorBalances); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &s.ValidatorRegistryLatestChangeSlot); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &s.ValidatorRegistryExitCount); err != nil {
		return err
	}
	s.ValidatorRegistryDeltaChainTip = chainhash.Hash{}
	if err := ssz.Decode(reader, &s.ValidatorRegistryDeltaChainTip); err != nil {
		return err
	}
	s.RandaoMix = chainhash.Hash{}
	if err := ssz.Decode(reader, &s.RandaoMix); err != nil {
		return err
	}
	s.NextSeed = chainhash.Hash{}
	if err := ssz.Decode(reader, &s.NextSeed); err != nil {
		return err
	}
	s.ShardAndCommitteeForSlots = [][]ShardAndCommittee{}
	if err := ssz.Decode(reader, &s.ShardAndCommitteeForSlots); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &s.PreviousJustifiedSlot); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &s.JustifiedSlot); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &s.JustificationBitfield); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &s.FinalizedSlot); err != nil {
		return err
	}
	s.LatestCrosslinks = []Crosslink{}
	if err := ssz.Decode(reader, &s.LatestCrosslinks); err != nil {
		return err
	}
	s.LatestBlockHashes = []chainhash.Hash{}
	if err := ssz.Decode(reader, &s.LatestBlockHashes); err != nil {
		return err
	}
	s.LatestPenalizedExitBalances = []uint64{}
	if err := ssz.Decode(reader, &s.LatestPenalizedExitBalances); err != nil {
		return err
	}
	s.LatestAttestations = []PendingAttestation{}
	if err := ssz.Decode(reader, &s.LatestAttestations); err != nil {
		return err
	}
	s.BatchedBlockRoots = []chainhash.Hash{}
	if err := ssz.Decode(reader, &s.BatchedBlockRoots); err != nil {
		return err
	}
	return nil
}

// TreeHashSSZ implements Hashable  in  github.com/prysmaticlabs/prysm/shared/ssz
func (s State) TreeHashSSZ() ([32]byte, error) {
	buf := bytes.Buffer{}
	s.EncodeSSZ(&buf)
	hash := chainhash.HashB(buf.Bytes())
	var result [32]byte
	copy(result[:], hash)
	return result, nil
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

// GetTotalBalance gets the total balance of the provided validator indices.
func (s *State) GetTotalBalance(activeValidators []uint32, c *config.Config) uint64 {
	balance := uint64(0)
	for _, i := range activeValidators {
		balance += s.GetEffectiveBalance(i, c)
	}
	return balance
}

// GetTotalBalanceMap gets the total balance of the provided validator indices with a map input.
func (s *State) GetTotalBalanceMap(activeValidators map[uint32]struct{}, c *config.Config) uint64 {
	balance := uint64(0)
	for i := range activeValidators {
		balance += s.GetEffectiveBalance(i, c)
	}
	return balance
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

	valid, err := bls.VerifySig(&pubkey, proofOfPossessionDataBytes, &proofOfPossession, bls.DomainDeposit)
	if err != nil {
		return false, err
	}
	return valid, nil
}

// ApplyProposerSlashing applies a proposer slashing if valid.
func (s *State) ApplyProposerSlashing(proposerSlashing ProposerSlashing, config *config.Config) error {
	if proposerSlashing.ProposerIndex >= uint32(len(s.ValidatorRegistry)) {
		return errors.New("invalid proposer index")
	}
	proposer := &s.ValidatorRegistry[proposerSlashing.ProposerIndex]
	if proposerSlashing.ProposalData1.Slot != proposerSlashing.ProposalData2.Slot {
		return errors.New("proposer slashing request does not have same slot")
	}
	if proposerSlashing.ProposalData1.Shard != proposerSlashing.ProposalData2.Shard {
		return errors.New("proposer slashing request does not have same shard number")
	}
	if proposerSlashing.ProposalData1.BlockHash.IsEqual(&proposerSlashing.ProposalData2.BlockHash) {
		return errors.New("proposer slashing request has same block hash (same proposal)")
	}
	if proposer.Status == ExitedWithPenalty {
		return errors.New("proposer is already exited")
	}
	hashProposal1, err := ssz.TreeHash(proposerSlashing.ProposalData1)
	if err != nil {
		return err
	}
	hashProposal2, err := ssz.TreeHash(proposerSlashing.ProposalData2)
	if err != nil {
		return err
	}
	valid, err := bls.VerifySig(&proposer.Pubkey, hashProposal2[:], &proposerSlashing.ProposalSignature2, bls.DomainProposal)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid proposer signature")
	}
	valid, err = bls.VerifySig(&proposer.Pubkey, hashProposal1[:], &proposerSlashing.ProposalSignature1, bls.DomainProposal)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid proposer signature")
	}
	return s.UpdateValidatorStatus(proposerSlashing.ProposerIndex, ExitedWithPenalty, config)
}

func indices(vote SlashableVoteData) []uint32 {
	return append(vote.AggregateSignaturePoC0Indices, vote.AggregateSignaturePoC1Indices...)
}

func isDoubleVote(ad1 AttestationData, ad2 AttestationData, c *config.Config) bool {
	targetEpoch1 := ad1.Slot / c.EpochLength
	targetEpoch2 := ad2.Slot / c.EpochLength
	return targetEpoch1 == targetEpoch2
}

func isSurroundVote(ad1 AttestationData, ad2 AttestationData, c *config.Config) bool {
	targetEpoch1 := ad1.Slot / c.EpochLength
	targetEpoch2 := ad2.Slot / c.EpochLength
	sourceEpoch1 := ad1.JustifiedSlot / c.EpochLength
	sourceEpoch2 := ad2.JustifiedSlot / c.EpochLength
	return (sourceEpoch1 < sourceEpoch2) &&
		(sourceEpoch2+1 == targetEpoch2) &&
		(targetEpoch2 < targetEpoch1)
}

// GetDomain gets the domain for a slot and type.
func GetDomain(forkData ForkData, slot uint64, domainType uint64) uint64 {
	return (forkData.GetVersionForSlot(slot) << 32) + domainType
}

func (s *State) verifySlashableVoteData(voteData SlashableVoteData, c *config.Config) bool {
	if len(voteData.AggregateSignaturePoC0Indices)+len(voteData.AggregateSignaturePoC1Indices) > int(c.MaxCasperVotes) {
		return false
	}

	pubKey0 := bls.NewAggregatePublicKey()
	pubKey1 := bls.NewAggregatePublicKey()

	for _, i := range voteData.AggregateSignaturePoC0Indices {
		pubKey0.AggregatePubKey(&s.ValidatorRegistry[i].Pubkey)
	}

	for _, i := range voteData.AggregateSignaturePoC1Indices {
		pubKey1.AggregatePubKey(&s.ValidatorRegistry[i].Pubkey)
	}

	ad0 := AttestationDataAndCustodyBit{voteData.Data, true}
	ad1 := AttestationDataAndCustodyBit{voteData.Data, true}

	ad0Hash, err := ssz.TreeHash(ad0)
	if err != nil {
		return false
	}

	ad1Hash, err := ssz.TreeHash(ad1)
	if err != nil {
		return false
	}

	return bls.VerifyAggregate([]*bls.PublicKey{
		pubKey0,
		pubKey1,
	}, [][]byte{
		ad0Hash[:],
		ad1Hash[:],
	}, &voteData.AggregateSignature, GetDomain(s.ForkData, s.Slot, bls.DomainAttestation))
}

// ApplyCasperSlashing applies a casper slashing claim to the current state.
func (s *State) ApplyCasperSlashing(casperSlashing CasperSlashing, c *config.Config) error {
	intersection := []uint32{}
	indices1 := indices(casperSlashing.Votes1)
	indices2 := indices(casperSlashing.Votes2)
	for _, k := range indices1 {
		for _, m := range indices2 {
			if k == m {
				intersection = append(intersection, k)
			}
		}
	}

	if len(intersection) == 0 {
		return errors.New("casper slashing does not include intersection")
	}

	if casperSlashing.Votes1.Data.Equals(&casperSlashing.Votes2.Data) {
		return errors.New("casper slashing votes are the same")
	}

	if !isDoubleVote(casperSlashing.Votes1.Data, casperSlashing.Votes2.Data, c) &&
		!isSurroundVote(casperSlashing.Votes1.Data, casperSlashing.Votes2.Data, c) {
		return errors.New("casper slashing is not double or surround vote")
	}

	if !s.verifySlashableVoteData(casperSlashing.Votes1, c) {
		return errors.New("casper slashing signature did not verify")
	}

	if !s.verifySlashableVoteData(casperSlashing.Votes2, c) {
		return errors.New("casper slashing signature did not verify")
	}

	for _, i := range intersection {
		if s.ValidatorRegistry[i].Status != ExitedWithPenalty {
			s.UpdateValidatorStatus(i, ExitedWithPenalty, c)
		}
	}

	return nil
}

var zeroHash = chainhash.Hash{}

// ApplyExit validates and applies an exit.
func (s *State) ApplyExit(exit Exit, config *config.Config) error {
	validator := s.ValidatorRegistry[exit.ValidatorIndex]
	if validator.Status != Active {
		return errors.New("validator with exit is not active")
	}

	if s.Slot < exit.Slot {
		return errors.New("exit is not yet valid")
	}

	valid, err := bls.VerifySig(&validator.Pubkey, zeroHash[:], &exit.Signature, bls.DomainExit)
	if err != nil {
		return err
	}

	if !valid {
		return errors.New("signature is not valid")
	}

	s.UpdateValidatorStatus(uint32(exit.ValidatorIndex), ActivePendingExit, config)

	return nil
}

// GetAttestationParticipants gets the indices of participants.
func (s *State) GetAttestationParticipants(data AttestationData, participationBitfield []byte, c *config.Config) ([]uint32, error) {
	shardCommittees := s.GetShardCommitteesAtSlot(data.Slot, c)
	var shardCommittee ShardAndCommittee
	for i := range shardCommittees {
		if shardCommittees[i].Shard == data.Shard {
			shardCommittee = shardCommittees[i]
		}
	}

	if len(participationBitfield) != int(math.Ceil(float64(len(shardCommittee.Committee))/8)) {
		return nil, errors.New("participation bitfield is of incorrect length")
	}

	participants := []uint32{}
	for i, validatorIndex := range shardCommittee.Committee {
		participationBit := (participationBitfield[i/8] >> (7 - (uint(i) % 8))) % 2
		if participationBit == 1 {
			participants = append(participants, validatorIndex)
		}
	}
	return participants, nil
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

// EncodeSSZ implements Encodable
func (v Validator) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, v.Pubkey); err != nil {
		return err
	}
	if err := ssz.Encode(writer, v.WithdrawalCredentials); err != nil {
		return err
	}
	if err := ssz.Encode(writer, v.RandaoCommitment); err != nil {
		return err
	}
	if err := ssz.Encode(writer, v.RandaoSkips); err != nil {
		return err
	}
	if err := ssz.Encode(writer, v.Balance); err != nil {
		return err
	}
	if err := ssz.Encode(writer, v.Status); err != nil {
		return err
	}
	if err := ssz.Encode(writer, v.LatestStatusChangeSlot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, v.ExitCount); err != nil {
		return err
	}
	if err := ssz.Encode(writer, v.PoCCommitment); err != nil {
		return err
	}
	if err := ssz.Encode(writer, v.LastPoCChangeSlot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, v.SecondLastPoCChangeSlot); err != nil {
		return err
	}
	return nil
}

// EncodeSSZSize implements Encodable
func (v Validator) EncodeSSZSize() (uint32, error) {
	var sizeOfpubkey, sizeOfwithdrawalCredentials, sizeOfrandaoCommitment,
		sizeOfrandaoSkips, sizeOfbalance, sizeOfstatus, sizeOflatestStatusChangeSlot,
		sizeOfexitCount, sizeOfpoCCommitment, sizeOflastPoCChangeSlot, sizeOfsecondLastPoCChangeSlot uint32
	var err error
	if sizeOfpubkey, err = ssz.EncodeSize(v.Pubkey); err != nil {
		return 0, err
	}
	if sizeOfwithdrawalCredentials, err = ssz.EncodeSize(v.WithdrawalCredentials); err != nil {
		return 0, err
	}
	if sizeOfrandaoCommitment, err = ssz.EncodeSize(v.RandaoCommitment); err != nil {
		return 0, err
	}
	if sizeOfrandaoSkips, err = ssz.EncodeSize(v.RandaoSkips); err != nil {
		return 0, err
	}
	if sizeOfbalance, err = ssz.EncodeSize(v.Balance); err != nil {
		return 0, err
	}
	if sizeOfstatus, err = ssz.EncodeSize(v.Status); err != nil {
		return 0, err
	}
	if sizeOflatestStatusChangeSlot, err = ssz.EncodeSize(v.LatestStatusChangeSlot); err != nil {
		return 0, err
	}
	if sizeOfexitCount, err = ssz.EncodeSize(v.ExitCount); err != nil {
		return 0, err
	}
	if sizeOfpoCCommitment, err = ssz.EncodeSize(v.PoCCommitment); err != nil {
		return 0, err
	}
	if sizeOflastPoCChangeSlot, err = ssz.EncodeSize(v.LastPoCChangeSlot); err != nil {
		return 0, err
	}
	if sizeOfsecondLastPoCChangeSlot, err = ssz.EncodeSize(v.SecondLastPoCChangeSlot); err != nil {
		return 0, err
	}
	return sizeOfpubkey + sizeOfwithdrawalCredentials + sizeOfrandaoCommitment +
		sizeOfrandaoSkips + sizeOfbalance + sizeOfstatus + sizeOflatestStatusChangeSlot +
		sizeOfexitCount + sizeOfpoCCommitment + sizeOflastPoCChangeSlot + sizeOfsecondLastPoCChangeSlot, nil
}

// DecodeSSZ implements Decodable
func (v Validator) DecodeSSZ(reader io.Reader) error {
	v.Pubkey = bls.PublicKey{}
	if err := ssz.Decode(reader, &v.Pubkey); err != nil {
		return err
	}
	v.WithdrawalCredentials = chainhash.Hash{}
	if err := ssz.Decode(reader, &v.WithdrawalCredentials); err != nil {
		return err
	}
	v.RandaoCommitment = chainhash.Hash{}
	if err := ssz.Decode(reader, &v.RandaoCommitment); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &v.RandaoSkips); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &v.Balance); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &v.Status); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &v.LatestStatusChangeSlot); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &v.ExitCount); err != nil {
		return err
	}
	v.PoCCommitment = chainhash.Hash{}
	if err := ssz.Decode(reader, &v.PoCCommitment); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &v.LastPoCChangeSlot); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &v.SecondLastPoCChangeSlot); err != nil {
		return err
	}
	return nil
}

// TreeHashSSZ implements Hashable  in  github.com/prysmaticlabs/prysm/shared/ssz
func (v Validator) TreeHashSSZ() ([32]byte, error) {
	buf := bytes.Buffer{}
	v.EncodeSSZ(&buf)
	hash := chainhash.HashB(buf.Bytes())
	var result [32]byte
	copy(result[:], hash)
	return result, nil
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
func GetActiveValidatorIndices(validators []Validator) []uint32 {
	active := []uint32{}
	for i, v := range validators {
		if v.IsActive() {
			active = append(active, uint32(i))
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

// EncodeSSZ implements Encodable
func (c Crosslink) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, c.Slot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, c.ShardBlockHash); err != nil {
		return err
	}

	return nil
}

// EncodeSSZSize implements Encodable
func (c Crosslink) EncodeSSZSize() (uint32, error) {
	var sizeOfSlot, sizeOfShardBlockHash uint32
	var err error
	if sizeOfSlot, err = ssz.EncodeSize(c.Slot); err != nil {
		return 0, err
	}
	if sizeOfShardBlockHash, err = ssz.EncodeSize(c.ShardBlockHash); err != nil {
		return 0, err
	}
	return sizeOfSlot + sizeOfShardBlockHash, nil
}

// DecodeSSZ implements Decodable
func (c Crosslink) DecodeSSZ(reader io.Reader) error {
	if err := ssz.Decode(reader, &c.Slot); err != nil {
		return err
	}
	c.ShardBlockHash = chainhash.Hash{}
	if err := ssz.Decode(reader, &c.ShardBlockHash); err != nil {
		return err
	}

	return nil
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

// EncodeSSZ implements Encodable
func (sc ShardAndCommittee) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, sc.Shard); err != nil {
		return err
	}
	if err := ssz.Encode(writer, sc.Committee); err != nil {
		return err
	}
	if err := ssz.Encode(writer, sc.TotalValidatorCount); err != nil {
		return err
	}

	return nil
}

// EncodeSSZSize implements Encodable
func (sc ShardAndCommittee) EncodeSSZSize() (uint32, error) {
	var sizeOShard, sizeOfCommittee, sizeOfTotalValidatorCount uint32
	var err error
	if sizeOShard, err = ssz.EncodeSize(sc.Shard); err != nil {
		return 0, err
	}
	if sizeOfCommittee, err = ssz.EncodeSize(sc.Committee); err != nil {
		return 0, err
	}
	if sizeOfTotalValidatorCount, err = ssz.EncodeSize(sc.TotalValidatorCount); err != nil {
		return 0, err
	}
	return sizeOShard + sizeOfCommittee + sizeOfTotalValidatorCount, nil
}

// DecodeSSZ implements Decodable
func (sc ShardAndCommittee) DecodeSSZ(reader io.Reader) error {
	if err := ssz.Decode(reader, &sc.Shard); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &sc.Committee); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &sc.TotalValidatorCount); err != nil {
		return err
	}

	return nil
}
