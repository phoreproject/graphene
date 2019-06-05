package primitives

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"

	"github.com/prysmaticlabs/prysm/shared/ssz"
)

// ValidatorRegistryDeltaBlock is a validator change hash.
type ValidatorRegistryDeltaBlock struct {
	LatestRegistryDeltaRoot chainhash.Hash
	ValidatorIndex          uint32
	Pubkey                  [96]byte
	Flag                    uint64
}

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

// Copy returns a copy of the fork data.
func (f ForkData) Copy() ForkData {
	return ForkData{PreForkVersion: f.PreForkVersion, PostForkVersion: f.PostForkVersion, ForkSlotNumber: f.ForkSlotNumber}
}

// ToProto gets the protobuf representation of the fork data.
func (f ForkData) ToProto() *pb.ForkData {
	return &pb.ForkData{
		PreForkVersion:  f.PreForkVersion,
		PostForkVersion: f.PostForkVersion,
		ForkSlot:        f.ForkSlotNumber,
	}
}

// ForkDataFromProto gets the fork data from the proto representation.
func ForkDataFromProto(data *pb.ForkData) (*ForkData, error) {
	return &ForkData{
		PreForkVersion:  data.PreForkVersion,
		PostForkVersion: data.PostForkVersion,
		ForkSlotNumber:  data.ForkSlot,
	}, nil
}

// State is the state of a beacon block
type State struct {
	// MISC ITEMS
	// Slot is the current slot.
	Slot uint64

	EpochIndex uint64

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
		EpochIndex:                        s.EpochIndex,
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

// ToProto gets the protobuf representation of the state.
func (s *State) ToProto() *pb.State {
	validatorRegistry := make([]*pb.Validator, len(s.ValidatorRegistry))
	shardCommittees := make([]*pb.ShardCommitteesForSlot, len(s.ShardAndCommitteeForSlots))
	latestCrosslinks := make([]*pb.Crosslink, len(s.LatestCrosslinks))
	latestBlockHashes := make([][]byte, len(s.LatestBlockHashes))
	latestAttestations := make([]*pb.PendingAttestation, len(s.LatestAttestations))
	batchedBlockRoots := make([][]byte, len(s.BatchedBlockRoots))

	for i := range validatorRegistry {
		validatorRegistry[i] = s.ValidatorRegistry[i].ToProto()
	}

	for i := range shardCommittees {
		committees := make([]*pb.ShardCommittee, len(s.ShardAndCommitteeForSlots[i]))
		for j := range committees {
			committees[j] = s.ShardAndCommitteeForSlots[i][j].ToProto()
		}

		shardCommittees[i] = &pb.ShardCommitteesForSlot{
			Committees: committees,
		}
	}

	for i := range latestCrosslinks {
		latestCrosslinks[i] = s.LatestCrosslinks[i].ToProto()
	}

	for i := range latestBlockHashes {
		latestBlockHashes[i] = s.LatestBlockHashes[i][:]
	}

	for i := range latestAttestations {
		latestAttestations[i] = s.LatestAttestations[i].ToProto()
	}

	for i := range batchedBlockRoots {
		batchedBlockRoots[i] = s.BatchedBlockRoots[i][:]
	}

	return &pb.State{
		Slot:                              s.Slot,
		GenesisTime:                       s.GenesisTime,
		EpochIndex:                        s.EpochIndex,
		ForkData:                          s.ForkData.ToProto(),
		ValidatorRegistry:                 validatorRegistry,
		ValidatorBalances:                 s.ValidatorBalances,
		ValidatorRegistryLatestChangeSlot: s.ValidatorRegistryLatestChangeSlot,
		ValidatorRegistryDeltaChainTip:    s.ValidatorRegistryDeltaChainTip[:],
		ValidatorRegistryExitCount:        s.ValidatorRegistryExitCount,
		RandaoMix:                         s.RandaoMix[:],
		NextSeed:                          s.NextSeed[:],
		ShardCommittees:                   shardCommittees,
		PreviousJustifiedSlot:             s.PreviousJustifiedSlot,
		JustifiedSlot:                     s.JustifiedSlot,
		JustificationBitField:             s.JustificationBitfield,
		FinalizedSlot:                     s.FinalizedSlot,
		LatestCrosslinks:                  latestCrosslinks,
		LatestBlockHashes:                 latestBlockHashes,
		LatestPenalizedExitBalances:       s.LatestPenalizedExitBalances,
		LatestAttestations:                latestAttestations,
		BatchedBlockRoots:                 batchedBlockRoots,
	}
}

// StateFromProto gets the state fromo the protobuf representation.
func StateFromProto(s *pb.State) (*State, error) {
	validatorRegistry := make([]Validator, len(s.ValidatorRegistry))
	shardAndCommitteeForSlots := make([][]ShardAndCommittee, len(s.ShardCommittees))
	latestCrosslinks := make([]Crosslink, len(s.LatestCrosslinks))
	latestBlockHashes := make([]chainhash.Hash, len(s.LatestBlockHashes))
	latestAttestations := make([]PendingAttestation, len(s.LatestAttestations))
	batchedBlockRoots := make([]chainhash.Hash, len(s.BatchedBlockRoots))

	fd, err := ForkDataFromProto(s.ForkData)
	if err != nil {
		return nil, err
	}

	for i := range validatorRegistry {
		v, err := ValidatorFromProto(s.ValidatorRegistry[i])
		if err != nil {
			return nil, err
		}
		validatorRegistry[i] = *v
	}

	for i := range shardAndCommitteeForSlots {
		shardAndCommitteeForSlots[i] = make([]ShardAndCommittee, len(s.ShardCommittees[i].Committees))
		for j := range shardAndCommitteeForSlots[i] {
			sc, err := ShardAndCommitteeFromProto(s.ShardCommittees[i].Committees[j])
			if err != nil {
				return nil, err
			}
			shardAndCommitteeForSlots[i][j] = *sc
		}
	}

	for i := range latestCrosslinks {
		c, err := CrosslinkFromProto(s.LatestCrosslinks[i])
		if err != nil {
			return nil, err
		}
		latestCrosslinks[i] = *c
	}

	for i := range latestBlockHashes {
		err := latestBlockHashes[i].SetBytes(s.LatestBlockHashes[i])
		if err != nil {
			return nil, err
		}
	}

	for i := range latestAttestations {
		a, err := PendingAttestationFromProto(s.LatestAttestations[i])
		if err != nil {
			return nil, err
		}
		latestAttestations[i] = *a
	}

	for i := range batchedBlockRoots {
		err := batchedBlockRoots[i].SetBytes(s.BatchedBlockRoots[i])
		if err != nil {
			return nil, err
		}
	}

	newState := &State{
		Slot:                              s.Slot,
		GenesisTime:                       s.GenesisTime,
		EpochIndex:                        s.EpochIndex,
		ForkData:                          *fd,
		ValidatorRegistry:                 validatorRegistry,
		ValidatorRegistryLatestChangeSlot: s.ValidatorRegistryLatestChangeSlot,
		ValidatorRegistryExitCount:        s.ValidatorRegistryExitCount,
		ShardAndCommitteeForSlots:         shardAndCommitteeForSlots,
		PreviousJustifiedSlot:             s.PreviousJustifiedSlot,
		JustifiedSlot:                     s.JustifiedSlot,
		JustificationBitfield:             s.JustificationBitField,
		FinalizedSlot:                     s.FinalizedSlot,
		LatestCrosslinks:                  latestCrosslinks,
		LatestBlockHashes:                 latestBlockHashes,
		LatestAttestations:                latestAttestations,
		BatchedBlockRoots:                 batchedBlockRoots,
	}

	err = newState.ValidatorRegistryDeltaChainTip.SetBytes(s.ValidatorRegistryDeltaChainTip)
	if err != nil {
		return nil, err
	}

	err = newState.RandaoMix.SetBytes(s.RandaoMix)
	if err != nil {
		return nil, err
	}

	err = newState.NextSeed.SetBytes(s.NextSeed)
	if err != nil {
		return nil, err
	}

	newState.ValidatorBalances = append([]uint64{}, s.ValidatorBalances...)
	newState.LatestPenalizedExitBalances = append([]uint64{}, s.LatestPenalizedExitBalances...)

	return newState, nil
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
func GetNewValidatorRegistryDeltaChainTip(currentValidatorRegistryDeltaChainTip chainhash.Hash, validatorIndex uint32, pubkey [96]byte, flag uint64) (chainhash.Hash, error) {
	return ssz.TreeHash(ValidatorRegistryDeltaBlock{
		LatestRegistryDeltaRoot: currentValidatorRegistryDeltaChainTip,
		ValidatorIndex:          validatorIndex,
		Pubkey:                  pubkey,
		Flag:                    flag,
	})
}

const (
	// ActivationFlag is a flag to indicate a validator is joining
	ActivationFlag = iota
	// ExitFlag is a flag to indeicate a validator is leaving
	ExitFlag
)

// ActivateValidator activates a validator in the state at a certain index.
func (s *State) ActivateValidator(index uint32) error {
	validator := &s.ValidatorRegistry[index]
	if validator.Status != PendingActivation {
		return errors.New("validator is not pending activation")
	}

	validator.Status = Active
	validator.LatestStatusChangeSlot = s.Slot
	deltaChainTip, err := GetNewValidatorRegistryDeltaChainTip(s.ValidatorRegistryDeltaChainTip, index, validator.Pubkey, ActivationFlag)
	if err != nil {
		return err
	}
	s.ValidatorRegistryDeltaChainTip = deltaChainTip
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
func (s *State) GetBeaconProposerIndex(slot uint64, c *config.Config) (uint32, error) {
	committees, err := s.GetShardCommitteesAtSlot(slot, c)
	if err != nil {
		return 0, err
	}
	firstCommittee := committees[0].Committee
	return firstCommittee[int(slot)%len(firstCommittee)], nil
}

// ExitValidator handles state changes when a validator exits.
func (s *State) ExitValidator(index uint32, status uint64, c *config.Config) error {
	validator := s.ValidatorRegistry[index]
	prevStatus := validator.Status

	if prevStatus == ExitedWithPenalty {
		return nil
	}

	validator.Status = status
	validator.LatestStatusChangeSlot = s.Slot

	if status == ExitedWithPenalty {
		s.LatestPenalizedExitBalances[s.Slot/c.CollectivePenaltyCalculationPeriod] += s.GetEffectiveBalance(index, c)

		whistleblowerIndex, err := s.GetBeaconProposerIndex(s.Slot, c)
		if err != nil {
			return err
		}
		whistleblowerReward := s.GetEffectiveBalance(index, c) / c.WhistleblowerRewardQuotient
		s.ValidatorBalances[whistleblowerIndex] += whistleblowerReward
		s.ValidatorBalances[index] -= whistleblowerReward
	}

	if prevStatus == ExitedWithoutPenalty {
		return nil
	}

	s.ValidatorRegistryExitCount++
	validator.ExitCount = s.ValidatorRegistryExitCount
	deltaChainTip, err := GetNewValidatorRegistryDeltaChainTip(s.ValidatorRegistryDeltaChainTip, index, validator.Pubkey, ExitFlag)
	if err != nil {
		return err
	}
	s.ValidatorRegistryDeltaChainTip = deltaChainTip
	return nil
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
		err := s.ExitValidator(index, status, c)
		return err
	}
	return nil
}

// UpdateValidatorRegistry updates the registry and updates validator pending activation or exit.
func (s *State) UpdateValidatorRegistry(c *config.Config) error {
	activeValidatorIndices := GetActiveValidatorIndices(s.ValidatorRegistry)
	totalBalance := s.GetTotalBalance(activeValidatorIndices, c)
	maxBalanceChurn := c.MaxDeposit
	if maxBalanceChurn < (totalBalance / (2 * c.MaxBalanceChurnQuotient)) {
		maxBalanceChurn = totalBalance / (2 * c.MaxBalanceChurnQuotient)
	}

	balanceChurn := uint64(0)
	for idx, validator := range s.ValidatorRegistry {
		index := uint32(idx)
		if validator.Status == PendingActivation && s.ValidatorBalances[index] >= c.MaxDeposit {
			balanceChurn += s.GetEffectiveBalance(index, c)
			if balanceChurn > maxBalanceChurn {
				break
			}

			err := s.UpdateValidatorStatus(index, Active, c)
			if err != nil {
				return err
			}
		}
	}

	balanceChurn = 0
	for idx, validator := range s.ValidatorRegistry {
		index := uint32(idx)
		if validator.Status == ActivePendingExit {
			balanceChurn += s.GetEffectiveBalance(index, c)
			if balanceChurn > maxBalanceChurn {
				break
			}

			err := s.UpdateValidatorStatus(index, ExitedWithoutPenalty, c)
			if err != nil {
				return err
			}
		}
	}

	periodIndex := s.Slot / c.CollectivePenaltyCalculationPeriod
	totalPenalties := s.LatestPenalizedExitBalances[periodIndex]
	if periodIndex >= 1 {
		totalPenalties += s.LatestPenalizedExitBalances[periodIndex-1]
	}
	if periodIndex >= 2 {
		totalPenalties += s.LatestPenalizedExitBalances[periodIndex-2]
	}

	for idx, validator := range s.ValidatorRegistry {
		index := uint32(idx)
		if validator.Status != ExitedWithPenalty {
			penaltyFactor := totalPenalties * 3
			if totalBalance < penaltyFactor {
				penaltyFactor = totalBalance
			}
			s.ValidatorBalances[index] -= s.GetEffectiveBalance(index, c) * penaltyFactor / totalBalance
		}
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

// GetShardCommitteesAtSlot gets the committees assigned to a specific slot.
func (s *State) GetShardCommitteesAtSlot(slot uint64, c *config.Config) ([]ShardAndCommittee, error) {
	stateSlot := s.EpochIndex * c.EpochLength
	earliestSlot := int64(stateSlot) - int64(stateSlot%c.EpochLength) - int64(c.EpochLength)
	if int64(slot)-earliestSlot < 0 || int64(slot)-earliestSlot >= int64(len(s.ShardAndCommitteeForSlots)) {
		return nil, errors.WithStack(fmt.Errorf("could not get slot %d when state is at slot %d", slot, stateSlot))
	}
	return s.ShardAndCommitteeForSlots[int64(slot)-earliestSlot], nil
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
	committees, err := s.GetShardCommitteesAtSlot(slot, con)
	if err != nil {
		return nil, err
	}
	return ShardCommitteeByShardID(shardID, committees)
}

// ValidateProofOfPossession validates a proof of possession for a new validator.
func (s *State) ValidateProofOfPossession(pubkey *bls.PublicKey, proofOfPossession bls.Signature, withdrawalCredentials chainhash.Hash) (bool, error) {
	// fixme

	h, err := ssz.TreeHash(pubkey.Serialize())
	if err != nil {
		return false, err
	}
	valid, err := bls.VerifySig(pubkey, h[:], &proofOfPossession, bls.DomainDeposit)
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
	pub, err := proposer.GetPublicKey()
	if err != nil {
		return err
	}

	sigProposal2, err := bls.DeserializeSignature(proposerSlashing.ProposalSignature2)
	if err != nil {
		return err
	}

	valid, err := bls.VerifySig(pub, hashProposal2[:], sigProposal2, bls.DomainProposal)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid proposer signature")
	}

	sigProposal1, err := bls.DeserializeSignature(proposerSlashing.ProposalSignature1)
	if err != nil {
		return err
	}

	valid, err = bls.VerifySig(pub, hashProposal1[:], sigProposal1, bls.DomainProposal)
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
		p, err := s.ValidatorRegistry[i].GetPublicKey()
		if err != nil {
			panic(err)
		}
		pubKey0.AggregatePubKey(p)
	}

	for _, i := range voteData.AggregateSignaturePoC1Indices {
		p, err := s.ValidatorRegistry[i].GetPublicKey()
		if err != nil {
			panic(err)
		}
		pubKey1.AggregatePubKey(p)
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

	aggregateSignature, err := bls.DeserializeSignature(voteData.AggregateSignature)
	if err != nil {
		panic(err)
	}

	return bls.VerifyAggregate([]*bls.PublicKey{
		pubKey0,
		pubKey1,
	}, [][]byte{
		ad0Hash[:],
		ad1Hash[:],
	}, aggregateSignature, GetDomain(s.ForkData, s.Slot, bls.DomainAttestation))
}

// ApplyCasperSlashing applies a casper slashing claim to the current state.
func (s *State) ApplyCasperSlashing(casperSlashing CasperSlashing, c *config.Config) error {
	var intersection []uint32
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
			err := s.UpdateValidatorStatus(i, ExitedWithPenalty, c)
			if err != nil {
				return err
			}
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

	validatorPub, err := validator.GetPublicKey()
	if err != nil {
		return err
	}

	exitSig, err := bls.DeserializeSignature(exit.Signature)
	if err != nil {
		return err
	}

	valid, err := bls.VerifySig(validatorPub, zeroHash[:], exitSig, bls.DomainExit)
	if err != nil {
		return err
	}

	if !valid {
		return errors.New("signature is not valid")
	}

	err = s.UpdateValidatorStatus(uint32(exit.ValidatorIndex), ActivePendingExit, config)
	if err != nil {
		return err
	}

	return nil
}

// GetAttestationParticipants gets the indices of participants.
func (s *State) GetAttestationParticipants(data AttestationData, participationBitfield []byte, c *config.Config) ([]uint32, error) {
	shardCommittees, err := s.GetShardCommitteesAtSlot(data.Slot, c)
	if err != nil {
		return nil, err
	}
	var shardCommittee ShardAndCommittee
	found := false
	for i := range shardCommittees {
		if shardCommittees[i].Shard == data.Shard {
			shardCommittee = shardCommittees[i]
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("could not find committee at slot %d and shard %d", data.Slot, data.Shard)
	}

	if len(participationBitfield) != (len(shardCommittee.Committee)+7)/8 {
		return nil, errors.New("participation bitfield is of incorrect length")
	}

	var participants []uint32
	for i, validatorIndex := range shardCommittee.Committee {
		participationBit := participationBitfield[i/8] & (1 << (uint(i) % 8))
		if participationBit != 0 {
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
func (s *State) ProcessDeposit(pubkey *bls.PublicKey, amount uint64, proofOfPossession [48]byte, withdrawalCredentials chainhash.Hash, skipValidation bool, c *config.Config) (uint32, error) {
	if !skipValidation {
		sig, err := bls.DeserializeSignature(proofOfPossession)
		if err != nil {
			return 0, err
		}

		sigValid, err := s.ValidateProofOfPossession(pubkey, *sig, withdrawalCredentials)
		if err != nil {
			return 0, err
		}
		if !sigValid {
			return 0, errors.New("invalid deposit signature")
		}
	}

	pubSer := pubkey.Serialize()

	validatorAlreadyRegisteredIndex := -1

	for i := range s.ValidatorRegistry {
		if bytes.Equal(s.ValidatorRegistry[i].Pubkey[:], pubSer[:]) {
			validatorAlreadyRegisteredIndex = i
		}
	}

	var index int

	if validatorAlreadyRegisteredIndex == -1 {
		validator := Validator{
			Pubkey:                  pubSer,
			XXXPubkeyCached:         pubkey,
			WithdrawalCredentials:   withdrawalCredentials,
			Status:                  PendingActivation,
			LatestStatusChangeSlot:  s.Slot,
			ExitCount:               0,
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

// ProcessSlot processes a single slot which should happen before the block transition and the epoch transition.
func (s *State) ProcessSlot(previousBlockRoot chainhash.Hash, c *config.Config) error {
	//slotTransitionTime := time.Now()

	// increase the slot number
	s.Slot++

	s.LatestBlockHashes[(s.Slot-1)%c.LatestBlockRootsLength] = previousBlockRoot

	if s.Slot%c.LatestBlockRootsLength == 0 {
		latestBlockHashesRoot, err := ssz.TreeHash(s.LatestBlockHashes)
		if err != nil {
			return err
		}
		s.BatchedBlockRoots = append(s.BatchedBlockRoots, latestBlockHashesRoot)
	}

	//slotTransitionDuration := time.Since(slotTransitionTime)

	//logrus.WithField("slot", s.Slot).WithField("duration", slotTransitionDuration).Info("slot transition")

	return nil
}

func intSqrt(n uint64) uint64 {
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// BlockView is an interface the provides access to blocks.
type BlockView interface {
	GetHashBySlot(slot uint64) (chainhash.Hash, error)
	Tip() (chainhash.Hash, error)
	SetTipSlot(slot uint64)
	GetLastStateRoot() (chainhash.Hash, error)
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

func clamp(min int, max int, val int) int {
	if val <= min {
		return min
	} else if val >= max {
		return max
	} else {
		return val
	}
}

// GetNewShuffling calculates the new shuffling of validators
// to slots and shards.
func GetNewShuffling(seed chainhash.Hash, validators []Validator, crosslinkingStart int, con *config.Config) [][]ShardAndCommittee {
	activeValidators := GetActiveValidatorIndices(validators)
	numActiveValidators := len(activeValidators)

	// clamp between 1 and b.config.ShardCount / b.config.EpochLength
	committeesPerSlot := clamp(1, con.ShardCount/int(con.EpochLength), numActiveValidators/int(con.EpochLength)/con.TargetCommitteeSize)

	output := make([][]ShardAndCommittee, con.EpochLength)

	shuffledValidatorIndices := ShuffleValidators(activeValidators, seed)

	validatorsPerSlot := Split(shuffledValidatorIndices, uint32(con.EpochLength))

	for slot, slotIndices := range validatorsPerSlot {
		shardIndices := Split(slotIndices, uint32(committeesPerSlot))

		shardIDStart := crosslinkingStart + slot*committeesPerSlot

		shardCommittees := make([]ShardAndCommittee, len(shardIndices))
		for shardPosition, indices := range shardIndices {
			shardCommittees[shardPosition] = ShardAndCommittee{
				Shard:     uint64((shardIDStart + shardPosition) % con.ShardCount),
				Committee: indices,
			}
		}

		output[slot] = shardCommittees
	}
	return output
}

// Receipt is a way of representing slashings or rewards.
type Receipt struct {
	Slot   uint64
	Type   uint8
	Amount int64
	Index  uint32
}

// ReceiptTypeToMeaning converts a receipt type to a meaningful string.
func ReceiptTypeToMeaning(t uint8) string {
	switch t {
	case AttestedToPreviousEpochJustifiedSlot:
		return "attested to previous epoch justified slot"
	case AttestedToPreviousEpochBoundaryHash:
		return "attested to previous epoch justified boundary hash"
	case AttestedToCorrectBlockHashInPreviousEpoch:
		return "attested to correct block hash in previous epoch"
	case AttestationInclusionDistanceReward:
		return "attestation inclusion distance reward"
	case DidNotAttestToCorrectBeaconBlock:
		return "did not attest to correct beacon block"
	case DidNotAttestToPreviousEpochBoundary:
		return "did not attest to correct boundary block"
	case DidNotAttestToPreviousJustifiedSlot:
		return "did not attest to correct justified slot"
	case InactivityPenalty:
		return "inactivity"
	case AttestationInclusionDistancePenalty:
		return "attestation inclusion distance penalty"
	case ProposerReward:
		return "proposer reward"
	case AttestationParticipationReward:
		return "participated in attestation"
	case AttestationNonparticipationPenalty:
		return "did not participate in attestation"
	}
	return "unknown type"
}

const (
	// AttestedToPreviousEpochJustifiedSlot is a reward when a validator attests to the previous epoch justified slot.
	AttestedToPreviousEpochJustifiedSlot = iota

	// AttestedToPreviousEpochBoundaryHash is a reward when a validator attests to the previous epoch boundary hash.
	AttestedToPreviousEpochBoundaryHash

	// AttestedToCorrectBlockHashInPreviousEpoch is a reward when a validator attests to the correct beacon block hash for their slot.
	AttestedToCorrectBlockHashInPreviousEpoch

	// AttestationInclusionDistanceReward is a reward for including attestations in beacon blocks.
	AttestationInclusionDistanceReward

	// DidNotAttestToCorrectBeaconBlock is a penalty for not attesting to the correct beacon block.
	DidNotAttestToCorrectBeaconBlock

	// DidNotAttestToPreviousEpochBoundary is a penalty for not attesting to the previous epoch boundary.
	DidNotAttestToPreviousEpochBoundary

	// DidNotAttestToPreviousJustifiedSlot is a penalty for not attesting to the previous justified slot.
	DidNotAttestToPreviousJustifiedSlot

	// InactivityPenalty is a penalty for being exited with penalty.
	InactivityPenalty

	// AttestationInclusionDistancePenalty is a penalty for not including attestations in beacon blocks.
	AttestationInclusionDistancePenalty

	// ProposerReward is a reward for the proposer of a beacon block.
	ProposerReward

	// AttestationParticipationReward is a reward for choosing the correct shard block hash.
	AttestationParticipationReward

	// AttestationNonparticipationPenalty is a penalty for not choosing the correct shard block hash.
	AttestationNonparticipationPenalty
)

// ProcessEpochTransition processes an epoch transition and modifies state.
func (s *State) ProcessEpochTransition(c *config.Config, view BlockView) ([]Receipt, error) {
	activeValidatorIndices := GetActiveValidatorIndices(s.ValidatorRegistry)
	totalBalance := s.GetTotalBalance(activeValidatorIndices, c)

	// currentEpochAttestations is any attestation that happened in the last epoch
	var currentEpochAttestations []PendingAttestation
	for _, a := range s.LatestAttestations {
		// slot is greater than last epoch slot and slot is less than current slot
		if s.Slot-c.EpochLength <= a.Data.Slot && a.Data.Slot < s.Slot {
			currentEpochAttestations = append(currentEpochAttestations, a)
		}
	}

	// previousEpochAttestations is any attestation in the last epoch
	var previousEpochAttestations []PendingAttestation
	for _, a := range s.LatestAttestations {
		if s.Slot-2*c.EpochLength <= a.Data.Slot && a.Data.Slot < s.Slot-c.EpochLength {
			previousEpochAttestations = append(previousEpochAttestations, a)
		}
	}

	// previousEpochAttesterIndices are all participants of attestations in the previous epoch
	previousEpochAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochAttestations {
		participants, err := s.GetAttestationParticipants(a.Data, a.ParticipationBitfield, c)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			previousEpochAttesterIndices[p] = struct{}{}
		}
	}

	// previousEpochJustifiedAttestations are any attestations in the previous epoch that have a
	// justified slot equal to the previous justified slot.
	var previousEpochJustifiedAttestations []PendingAttestation
	for _, a := range previousEpochAttestations {
		if a.Data.JustifiedSlot == s.PreviousJustifiedSlot {
			previousEpochJustifiedAttestations = append(previousEpochJustifiedAttestations, a)
		}
	}
	for _, a := range currentEpochAttestations {
		if a.Data.JustifiedSlot == s.PreviousJustifiedSlot {
			previousEpochJustifiedAttestations = append(previousEpochJustifiedAttestations, a)
		}
	}

	// previousEpochJustifiedAttesterIndices are all participants of attestations in the previous
	// epoch with a justified slot equal to the previous justified slot.
	previousEpochJustifiedAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochJustifiedAttestations {
		participants, err := s.GetAttestationParticipants(a.Data, a.ParticipationBitfield, c)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			previousEpochJustifiedAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochJustifiedAttestingBalance := s.GetTotalBalanceMap(previousEpochJustifiedAttesterIndices, c)

	epochBoundaryHashMinus2 := chainhash.Hash{}
	if s.Slot >= 2*c.EpochLength {
		ebhm2, err := view.GetHashBySlot(s.Slot - 2*c.EpochLength)
		if err != nil {
			ebhm2 = chainhash.Hash{}
		}
		epochBoundaryHashMinus2 = ebhm2
	}

	epochBoundaryHashMinus1 := chainhash.Hash{}
	if s.Slot >= c.EpochLength {
		ebhm1, err := view.GetHashBySlot(s.Slot - c.EpochLength)
		if err != nil {
			ebhm1 = chainhash.Hash{}
		}
		epochBoundaryHashMinus1 = ebhm1
	}

	// previousEpochBoundaryAttestations is any attestation in the previous epoch where the epoch boundary is
	// set to the epoch boundary two epochs ago
	var previousEpochBoundaryAttestations []PendingAttestation
	for _, a := range previousEpochJustifiedAttestations {
		if epochBoundaryHashMinus2.IsEqual(&a.Data.EpochBoundaryHash) {
			previousEpochBoundaryAttestations = append(previousEpochBoundaryAttestations, a)
		}
	}

	// currentEpochBoundaryAttestations is any attestation in the previous epoch where the epoch boundary is
	// set to the epoch boundary two epochs ago
	var currentEpochBoundaryAttestations []PendingAttestation
	for _, a := range currentEpochAttestations {
		if epochBoundaryHashMinus1.IsEqual(&a.Data.EpochBoundaryHash) {
			currentEpochBoundaryAttestations = append(currentEpochBoundaryAttestations, a)
		}
	}

	previousEpochBoundaryAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochBoundaryAttestations {
		participants, err := s.GetAttestationParticipants(a.Data, a.ParticipationBitfield, c)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			previousEpochBoundaryAttesterIndices[p] = struct{}{}
		}
	}

	currentEpochBoundaryAttesterIndices := map[uint32]struct{}{}
	for _, a := range currentEpochBoundaryAttestations {
		participants, err := s.GetAttestationParticipants(a.Data, a.ParticipationBitfield, c)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			currentEpochBoundaryAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochBoundaryAttestingBalance := s.GetTotalBalanceMap(previousEpochBoundaryAttesterIndices, c)
	currentEpochBoundaryAttestingBalance := s.GetTotalBalanceMap(currentEpochBoundaryAttesterIndices, c)

	var previousEpochHeadAttestations []PendingAttestation
	for _, a := range previousEpochAttestations {
		blockRoot, err := view.GetHashBySlot(a.Data.Slot)
		if err != nil {
			break
		}
		if a.Data.BeaconBlockHash.IsEqual(&blockRoot) {
			previousEpochHeadAttestations = append(previousEpochHeadAttestations, a)
		}
	}

	previousEpochHeadAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochHeadAttestations {
		participants, err := s.GetAttestationParticipants(a.Data, a.ParticipationBitfield, c)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			previousEpochHeadAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochHeadAttestingBalance := s.GetTotalBalanceMap(previousEpochHeadAttesterIndices, c)

	oldPreviousJustifiedSlot := s.PreviousJustifiedSlot

	s.PreviousJustifiedSlot = s.JustifiedSlot
	s.JustificationBitfield = s.JustificationBitfield * 2

	if 3*previousEpochBoundaryAttestingBalance >= 2*totalBalance {
		s.JustificationBitfield |= (1 << 1) // mark the last epoch as justified
		s.JustifiedSlot = s.Slot - 2*c.EpochLength
	}
	if 3*currentEpochBoundaryAttestingBalance >= 2*totalBalance {
		s.JustificationBitfield |= (1 << 0) // mark the last epoch as justified
		s.JustifiedSlot = s.Slot - c.EpochLength
	}

	if ((s.JustificationBitfield>>1)%8 == 7 && s.PreviousJustifiedSlot == s.Slot-c.EpochLength*3) || // 1110
		((s.JustificationBitfield>>1)%4 == 3 && s.PreviousJustifiedSlot == s.Slot-c.EpochLength*2) { // 110 <- old previous justified would be
		s.FinalizedSlot = oldPreviousJustifiedSlot
	}

	if ((s.JustificationBitfield>>0)%8 == 7 && s.JustifiedSlot == s.Slot-c.EpochLength*2) ||
		((s.JustificationBitfield>>0)%4 == 3 && s.JustifiedSlot == s.Slot-c.EpochLength*1) {
		s.FinalizedSlot = s.PreviousJustifiedSlot
	}

	// attestingValidatorIndices gets the participants that attested to a certain shardblockRoot for a certain shardCommittee
	attestingValidatorIndices := func(shardComittee ShardAndCommittee, shardBlockRoot chainhash.Hash) ([]uint32, error) {
		outMap := map[uint32]struct{}{}
		for _, a := range currentEpochAttestations {
			if a.Data.Shard == shardComittee.Shard && a.Data.ShardBlockHash.IsEqual(&shardBlockRoot) {
				for i, s := range shardComittee.Committee {
					bit := a.ParticipationBitfield[i/8] & (1 << uint(i%8))
					if bit != 0 {
						outMap[s] = struct{}{}
					}
				}
			}
		}
		for _, a := range previousEpochAttestations {
			if a.Data.Shard == shardComittee.Shard && a.Data.ShardBlockHash.IsEqual(&shardBlockRoot) {
				for i, s := range shardComittee.Committee {
					bit := a.ParticipationBitfield[i/8] & (1 << uint(i%8))
					if bit != 0 {
						outMap[s] = struct{}{}
					}
				}
			}
		}
		out := make([]uint32, len(outMap))
		i := 0
		for id := range outMap {
			out[i] = id
			i++
		}
		return out, nil
	}

	// winningRoot finds the winning shard block Hash
	winningRoot := func(shardCommittee ShardAndCommittee) (*chainhash.Hash, error) {
		// find all possible hashes for the winning root
		possibleHashes := map[chainhash.Hash]struct{}{}
		for _, a := range currentEpochAttestations {
			if a.Data.Shard != shardCommittee.Shard {
				continue
			}
			possibleHashes[a.Data.ShardBlockHash] = struct{}{}
		}
		for _, a := range previousEpochAttestations {
			if a.Data.Shard != shardCommittee.Shard {
				continue
			}
			possibleHashes[a.Data.ShardBlockHash] = struct{}{}
		}

		if len(possibleHashes) == 0 {
			return nil, nil
		}

		// find the top balance
		topBalance := uint64(0)
		topHash := chainhash.Hash{}

		for b := range possibleHashes {
			validatorIndices, err := attestingValidatorIndices(shardCommittee, b)
			if err != nil {
				return nil, err
			}

			sumBalance := s.GetTotalBalance(validatorIndices, c)

			if sumBalance > totalBalance {
				topHash = b
				topBalance = sumBalance
			}

			if sumBalance == totalBalance {
				// in case of a tie, favor the lower hash
				if bytes.Compare(topHash[:], b[:]) > 0 {
					topHash = b
				}
			}
		}
		if topBalance == 0 {
			return nil, nil
		}
		return &topHash, nil
	}

	slotWinners := make([]map[uint64]chainhash.Hash, len(s.ShardAndCommitteeForSlots))

	for i, shardCommitteeAtSlot := range s.ShardAndCommitteeForSlots {
		for _, shardCommittee := range shardCommitteeAtSlot {
			bestRoot, err := winningRoot(shardCommittee)
			if err != nil {
				return nil, err
			}
			if bestRoot == nil {
				continue
			}
			if slotWinners[i] == nil {
				slotWinners[i] = make(map[uint64]chainhash.Hash)
			}
			slotWinners[i][shardCommittee.Shard] = *bestRoot
			attestingCommittee, err := attestingValidatorIndices(shardCommittee, *bestRoot)
			if err != nil {
				return nil, err
			}
			totalAttestingBalance := s.GetTotalBalance(attestingCommittee, c)
			totalBalance := s.GetTotalBalance(shardCommittee.Committee, c)

			if 3*totalAttestingBalance >= 2*totalBalance {
				s.LatestCrosslinks[shardCommittee.Shard] = Crosslink{
					Slot:           s.Slot,
					ShardBlockHash: *bestRoot,
				}
				logrus.WithFields(logrus.Fields{
					"slot":             s.Slot,
					"shardBlockHash":   bestRoot.String(),
					"totalAttestation": totalAttestingBalance,
					"totalBalance":     totalBalance,
				}).Debug("crosslink created")
			}
		}
	}

	// baseRewardQuotient = NetworkRewardQuotient * sqrt(balances)
	baseRewardQuotient := c.BaseRewardQuotient * intSqrt(totalBalance/config.UnitInCoin)

	baseReward := func(index uint32) uint64 {
		return s.GetEffectiveBalance(index, c) / baseRewardQuotient / 5
	}

	inactivityPenalty := func(index uint32, epochsSinceFinality uint64) uint64 {
		return s.GetEffectiveBalance(index, c) * epochsSinceFinality / c.InactivityPenaltyQuotient
	}

	epochsSinceFinality := (s.Slot - s.FinalizedSlot) / c.EpochLength

	// previousAttestationCache tracks which validator attested to which attestation
	previousAttestationCache := map[uint32]*PendingAttestation{}
	for _, a := range previousEpochAttestations {
		participation, err := s.GetAttestationParticipants(a.Data, a.ParticipationBitfield, c)
		if err != nil {
			return nil, err
		}

		for _, p := range participation {
			previousAttestationCache[p] = &a
		}
	}

	totalPenalized := uint64(0)
	totalRewarded := uint64(0)

	var receipts []Receipt

	// any validator not in previous_epoch_head_attester_indices is slashed
	// any validator not in previous_epoch_boundary_attester_indices is slashed
	// any validator not in previous_epoch_justified_attester_indices is slashed

	if s.Slot >= 2*c.EpochLength {
		for idx, validator := range s.ValidatorRegistry {
			index := uint32(idx)
			if validator.Status != Active {
				continue
			}

			// reward or slash based on if they attested to the correct head
			if _, found := previousEpochHeadAttesterIndices[index]; found {
				reward := baseReward(index) * previousEpochHeadAttestingBalance / totalBalance
				totalRewarded += reward
				s.ValidatorBalances[index] += reward
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   AttestedToCorrectBlockHashInPreviousEpoch,
					Index:  index,
					Amount: int64(reward),
				})
			} else {
				penalty := baseReward(index)
				totalPenalized += penalty
				s.ValidatorBalances[index] -= penalty
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   DidNotAttestToCorrectBeaconBlock,
					Index:  index,
					Amount: -int64(penalty),
				})
			}

			// reward or slash based on if they attested to the correct source
			if _, found := previousEpochBoundaryAttesterIndices[index]; found {
				reward := baseReward(index) * previousEpochBoundaryAttestingBalance / totalBalance
				totalRewarded += reward
				s.ValidatorBalances[index] += reward
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   AttestedToPreviousEpochBoundaryHash,
					Index:  index,
					Amount: int64(reward),
				})
			} else {
				penalty := baseReward(index)
				totalPenalized += baseReward(index)
				s.ValidatorBalances[index] -= baseReward(index)
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   DidNotAttestToPreviousEpochBoundary,
					Index:  index,
					Amount: -int64(penalty),
				})
			}

			// reward or slash based on if they attested to the correct target
			if _, found := previousEpochJustifiedAttesterIndices[index]; found {
				reward := baseReward(index) * previousEpochJustifiedAttestingBalance / totalBalance
				totalRewarded += reward
				s.ValidatorBalances[index] += reward
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   AttestedToPreviousEpochJustifiedSlot,
					Index:  index,
					Amount: int64(reward),
				})
			} else {
				penalty := baseReward(index)
				totalPenalized += penalty
				s.ValidatorBalances[index] -= penalty
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   DidNotAttestToPreviousJustifiedSlot,
					Index:  index,
					Amount: -int64(penalty),
				})
			}
		}
	}

	// after 4 epochs, start slashing all validators
	if epochsSinceFinality >= 4 {
		// any validator not in previous_epoch_head_attester_indices is slashed
		for idx, validator := range s.ValidatorRegistry {
			index := uint32(idx)
			if validator.Status == Active {
				penalty := baseReward(index) * 5
				totalPenalized += penalty
				s.ValidatorBalances[index] -= penalty
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   InactivityPenalty,
					Index:  index,
					Amount: -int64(penalty),
				})

				if _, found := previousEpochAttesterIndices[index]; !found {
					penalty := inactivityPenalty(index, epochsSinceFinality)
					totalPenalized += penalty
					s.ValidatorBalances[index] -= penalty
					receipts = append(receipts, Receipt{
						Slot:   s.Slot,
						Type:   InactivityPenalty,
						Index:  index,
						Amount: -int64(penalty),
					})
				}
			}
		}
	}

	// reward for including each attestation
	for attester := range previousEpochAttesterIndices {
		att := previousAttestationCache[attester]

		proposerIndex := att.ProposerIndex

		reward := baseReward(proposerIndex) / c.IncluderRewardQuotient
		totalRewarded += reward
		s.ValidatorBalances[proposerIndex] += reward

		receipts = append(receipts, Receipt{
			Slot:   s.Slot,
			Type:   ProposerReward,
			Index:  proposerIndex,
			Amount: int64(reward),
		})

		inclusionDistance := att.InclusionDelay

		reward = baseReward(attester) * c.MinAttestationInclusionDelay / inclusionDistance
		totalRewarded += reward
		s.ValidatorBalances[attester] += reward

		receipts = append(receipts, Receipt{
			Slot:   s.Slot,
			Type:   AttestationInclusionDistanceReward,
			Index:  attester,
			Amount: int64(reward),
		})
	}

	// these are penalties/rewards for voting against/for a winning crosslink
	if s.Slot >= 2*c.EpochLength {
		for slot, shardCommitteeAtSlot := range s.ShardAndCommitteeForSlots[:c.EpochLength] {
			for _, shardCommittee := range shardCommitteeAtSlot {
				winningRoot := slotWinners[slot][shardCommittee.Shard]
				participationIndices, err := attestingValidatorIndices(shardCommittee, winningRoot)
				if err != nil {
					return nil, err
				}

				participationIndicesMap := map[uint32]struct{}{}
				for _, p := range participationIndices {
					participationIndicesMap[p] = struct{}{}
				}

				totalAttestingBalance := s.GetTotalBalance(participationIndices, c)
				totalBalance := s.GetTotalBalance(shardCommittee.Committee, c)

				for _, index := range shardCommittee.Committee {
					if _, found := participationIndicesMap[index]; found {
						reward := baseReward(index) * totalAttestingBalance / totalBalance
						totalRewarded += reward
						s.ValidatorBalances[index] += reward
						receipts = append(receipts, Receipt{
							Slot:   s.Slot,
							Type:   AttestationParticipationReward,
							Index:  index,
							Amount: int64(reward),
						})
					} else {
						penalty := baseReward(index)
						totalPenalized += penalty
						s.ValidatorBalances[index] -= penalty
						receipts = append(receipts, Receipt{
							Slot:   s.Slot,
							Type:   AttestationNonparticipationPenalty,
							Index:  index,
							Amount: -int64(penalty),
						})
					}
				}
			}
		}
	}

	if err := s.exitValidatorsUnderMinimum(c); err != nil {
		return nil, err
	}

	shouldUpdateRegistry := s.shouldUpdateRegistry()

	// update registry if:
	// - a slot has been finalized after the last time the registry was changed
	// - every shard
	if shouldUpdateRegistry {
		err := s.UpdateValidatorRegistry(c)
		if err != nil {
			return nil, err
		}

		s.ValidatorRegistryLatestChangeSlot = s.Slot
		copy(s.ShardAndCommitteeForSlots[:c.EpochLength], s.ShardAndCommitteeForSlots[c.EpochLength:])
		lastSlot := s.ShardAndCommitteeForSlots[len(s.ShardAndCommitteeForSlots)-1]
		lastCommittee := lastSlot[len(lastSlot)-1]
		nextStartShard := (lastCommittee.Shard + 1) % uint64(c.ShardCount)
		newShuffling := GetNewShuffling(s.RandaoMix, s.ValidatorRegistry, int(nextStartShard), c)
		copy(s.ShardAndCommitteeForSlots[c.EpochLength:], newShuffling)
	} else {
		copy(s.ShardAndCommitteeForSlots[:c.EpochLength], s.ShardAndCommitteeForSlots[c.EpochLength:])
		epochsSinceLastRegistryChange := (s.Slot - s.ValidatorRegistryLatestChangeSlot) / c.EpochLength
		startShard := s.ShardAndCommitteeForSlots[0][0].Shard

		// epochsSinceLastRegistryChange is a power of 2
		if epochsSinceLastRegistryChange&(epochsSinceLastRegistryChange-1) == 0 {
			newShuffling := GetNewShuffling(s.RandaoMix, s.ValidatorRegistry, int(startShard), c)
			copy(s.ShardAndCommitteeForSlots[c.EpochLength:], newShuffling)
		}
	}
	s.EpochIndex = s.Slot / c.EpochLength

	newLatestAttestations := make([]PendingAttestation, 0)
	for _, a := range s.LatestAttestations {
		if a.Data.Slot >= s.Slot-2*c.EpochLength {
			newLatestAttestations = append(newLatestAttestations, a)
		}
	}

	s.LatestAttestations = newLatestAttestations

	return receipts, nil
}

func (s *State) exitValidatorsUnderMinimum(c *config.Config) error {
	for index, validator := range s.ValidatorRegistry {
		if validator.Status == Active && s.ValidatorBalances[index] < c.EjectionBalance {
			err := s.UpdateValidatorStatus(uint32(index), ExitedWithoutPenalty, c)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ValidateAttestation checks if the attestation is valid.
func (s *State) ValidateAttestation(att Attestation, verifySignature bool, view BlockView, c *config.Config) error {
	expectedJustifiedSlot := s.JustifiedSlot
	attEpoch := (att.Data.Slot - 1) / c.EpochLength
	if attEpoch < s.EpochIndex { // 8 -> 0, 9 -> 8
		expectedJustifiedSlot = s.PreviousJustifiedSlot
	}

	if att.Data.JustifiedSlot != expectedJustifiedSlot {
		return fmt.Errorf("justified slot did not match expected justified slot. (expected: %d, got: %d)", expectedJustifiedSlot, att.Data.JustifiedSlot)
	}

	node, err := view.GetHashBySlot(att.Data.JustifiedSlot)
	if err != nil {
		return err
	}

	if !att.Data.JustifiedBlockHash.IsEqual(&node) {
		return errors.New("justified block Hash did not match")
	}

	if len(s.LatestCrosslinks) <= int(att.Data.Shard) {
		return errors.New("invalid shard number")
	}

	latestCrosslinkRoot := s.LatestCrosslinks[att.Data.Shard].ShardBlockHash

	if !att.Data.LatestCrosslinkHash.IsEqual(&latestCrosslinkRoot) && !att.Data.ShardBlockHash.IsEqual(&latestCrosslinkRoot) {
		return errors.New("latest crosslink is invalid")
	}

	if verifySignature {
		participants, err := s.GetAttestationParticipants(att.Data, att.ParticipationBitfield, c)
		if err != nil {
			return err
		}

		dataRoot, err := ssz.TreeHash(AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
		if err != nil {
			return err
		}

		groupPublicKey := bls.NewAggregatePublicKey()
		for _, p := range participants {
			pub, err := s.ValidatorRegistry[p].GetPublicKey()
			if err != nil {
				return err
			}
			groupPublicKey.AggregatePubKey(pub)
		}

		aggSig, err := bls.DeserializeSignature(att.AggregateSig)
		if err != nil {
			return err
		}

		valid, err := bls.VerifySig(groupPublicKey, dataRoot[:], aggSig, GetDomain(s.ForkData, att.Data.Slot, bls.DomainAttestation))
		if err != nil {
			return err
		}

		if !valid {
			return errors.New("attestation signature is invalid")
		}
	}

	node, err = view.GetHashBySlot(att.Data.Slot)
	if err != nil {
		return err
	}

	if !att.Data.BeaconBlockHash.IsEqual(&node) {
		return fmt.Errorf("beacon block hash is invalid (expected: %s, got: %s)", node, att.Data.BeaconBlockHash)
	}

	if !att.Data.ShardBlockHash.IsEqual(&zeroHash) {
		return errors.New("invalid block Hash")
	}

	return nil
}

// applyAttestation verifies and applies an attestation to the given state.
func (s *State) applyAttestation(att Attestation, c *config.Config, view BlockView, verifySignature bool, proposerIndex uint32) error {
	err := s.ValidateAttestation(att, verifySignature, view, c)
	if err != nil {
		return err
	}

	// these checks are dependent on when the attestation is included
	if att.Data.Slot+c.MinAttestationInclusionDelay > s.Slot {
		return errors.New("attestation included too soon")
	}

	if att.Data.Slot+c.EpochLength < s.Slot {
		return errors.New("attestation was not included within 1 epoch")
	}

	s.LatestAttestations = append(s.LatestAttestations, PendingAttestation{
		Data:                  att.Data,
		ParticipationBitfield: att.ParticipationBitfield,
		CustodyBitfield:       att.CustodyBitfield,
		InclusionDelay:        s.Slot - att.Data.Slot,
		ProposerIndex:         proposerIndex,
	})

	return nil
}

// ProcessBlock tries to apply a block to the state.
func (s *State) ProcessBlock(block *Block, con *config.Config, view BlockView, verifySignature bool) error {

	//blockTransitionStart := time.Now()

	proposerIndex, err := s.GetBeaconProposerIndex(block.BlockHeader.SlotNumber-1, con)
	if err != nil {
		return err
	}

	if block.BlockHeader.SlotNumber != s.Slot {
		return fmt.Errorf("block has incorrect slot number (expecting: %d, got: %d)", s.Slot, block.BlockHeader.SlotNumber)
	}

	blockWithoutSignature := block.Copy()
	blockWithoutSignature.BlockHeader.Signature = bls.EmptySignature.Serialize()
	blockWithoutSignatureRoot, err := ssz.TreeHash(blockWithoutSignature)
	if err != nil {
		return err
	}

	proposal := ProposalSignedData{
		Slot:      s.Slot,
		Shard:     con.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot, err := ssz.TreeHash(proposal)
	if err != nil {
		return err
	}

	proposerPub, err := s.ValidatorRegistry[proposerIndex].GetPublicKey()
	if err != nil {
		return err
	}

	proposerSig, err := bls.DeserializeSignature(block.BlockHeader.Signature)
	if err != nil {
		return err
	}

	// process block and randao verifications concurrently

	if verifySignature {
		verificationResult := make(chan error)

		go func() {
			valid, err := bls.VerifySig(proposerPub, proposalRoot[:], proposerSig, bls.DomainProposal)
			if err != nil {
				verificationResult <- err
			}

			if !valid {
				verificationResult <- fmt.Errorf("block had invalid signature (expected signature from validator %d)", proposerIndex)
			}

			verificationResult <- nil
		}()

		var slotBytes [8]byte
		binary.BigEndian.PutUint64(slotBytes[:], block.BlockHeader.SlotNumber)
		slotBytesHash := chainhash.HashH(slotBytes[:])

		randaoSig, err := bls.DeserializeSignature(block.BlockHeader.RandaoReveal)
		if err != nil {
			return err
		}

		go func() {
			valid, err := bls.VerifySig(proposerPub, slotBytesHash[:], randaoSig, bls.DomainRandao)
			if err != nil {
				verificationResult <- err
			}
			if !valid {
				verificationResult <- errors.New("block has invalid randao signature")
			}

			verificationResult <- nil
		}()

		result1 := <-verificationResult
		result2 := <-verificationResult

		if result1 != nil {
			return result1
		}

		if result2 != nil {
			return result2
		}
	}

	randaoRevealSerialized, err := ssz.TreeHash(block.BlockHeader.RandaoReveal)
	if err != nil {
		return err
	}

	for i := range s.RandaoMix {
		s.RandaoMix[i] ^= randaoRevealSerialized[i]
	}

	if len(block.BlockBody.ProposerSlashings) > con.MaxProposerSlashings {
		return errors.New("more than maximum proposer slashings")
	}

	if len(block.BlockBody.CasperSlashings) > con.MaxCasperSlashings {
		return errors.New("more than maximum casper slashings")
	}

	if len(block.BlockBody.Attestations) > con.MaxAttestations {
		return errors.New("more than maximum attestations")
	}

	if len(block.BlockBody.Exits) > con.MaxExits {
		return errors.New("more than maximum exits")
	}

	if len(block.BlockBody.Deposits) > con.MaxDeposits {
		return errors.New("more than maximum deposits")
	}

	for _, slashing := range block.BlockBody.ProposerSlashings {
		err := s.ApplyProposerSlashing(slashing, con)
		if err != nil {
			return err
		}
	}

	for _, c := range block.BlockBody.CasperSlashings {
		err := s.ApplyCasperSlashing(c, con)
		if err != nil {
			return err
		}
	}

	for _, a := range block.BlockBody.Attestations {
		err := s.applyAttestation(a, con, view, verifySignature, proposerIndex)
		if err != nil {
			return err
		}
	}

	// process deposits here

	for _, e := range block.BlockBody.Exits {
		err := s.ApplyExit(e, con)
		if err != nil {
			return err
		}
	}

	//blockTransitionTime := time.Since(blockTransitionStart)

	//logrus.WithField("slot", s.Slot).WithField("block", block.BlockHeader.SlotNumber).WithField("duration", blockTransitionTime).Info("block transition")

	// Check state root.
	expectedStateRoot, err := view.GetLastStateRoot()
	if err != nil {
		return err
	}
	if !block.BlockHeader.StateRoot.IsEqual(&expectedStateRoot) {
		return fmt.Errorf("state root doesn't match (expected: %s, got: %s)", expectedStateRoot, block.BlockHeader.StateRoot)
	}

	//blockTransitionTime := time.Since(blockTransitionStart)

	//logrus.WithField("slot", s.Slot).WithField("block", block.BlockHeader.SlotNumber).WithField("duration", blockTransitionTime).Info("block transition")

	return nil
}

// ProcessSlots uses the current head to process slots up to a certain slot, applying
// slot transitions and epoch transitions, and returns the updated state. Note that this should
// only process up to the current slot number so that the lastBlockHash remains constant.
func (s *State) ProcessSlots(upTo uint64, view BlockView, c *config.Config) ([]Receipt, error) {
	var receipts []Receipt

	for s.Slot < upTo {
		// this only happens when there wasn't a block at the first slot of the epoch
		if s.Slot/c.EpochLength > s.EpochIndex && s.Slot%c.EpochLength == 0 {
			//t := time.Now()

			epochReceipts, err := s.ProcessEpochTransition(c, view)
			if err != nil {
				return nil, err
			}

			receipts = append(receipts, epochReceipts...)
			//logrus.WithField("time", time.Since(t)).Debug("done processing epoch transition")
		}

		tip, err := view.Tip()
		if err != nil {
			return nil, err
		}

		err = s.ProcessSlot(tip, c)
		if err != nil {
			return nil, err
		}

		view.SetTipSlot(s.Slot)
	}

	return receipts, nil
}

// shouldUpdateRegistry returns if the validator registry is ready to be shuffled.
// The registry should only be updated if every shard has created a crosslink since the
// last update and the beacon chain has been finalized since the last update.
func (s *State) shouldUpdateRegistry() bool {
	if s.FinalizedSlot <= s.ValidatorRegistryLatestChangeSlot {
		return false
	}

	for _, shardAndCommittees := range s.ShardAndCommitteeForSlots {
		for _, committee := range shardAndCommittees {
			if s.LatestCrosslinks[committee.Shard].Slot <= s.ValidatorRegistryLatestChangeSlot {
				return false
			}
		}
	}

	return true
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
	Pubkey [96]byte
	// XXXPubkeyCached is the cached deserialized public key.
	XXXPubkeyCached *bls.PublicKey
	// Withdrawal credentials
	WithdrawalCredentials chainhash.Hash
	// Status code
	Status uint64
	// Slot when validator last changed status (or 0)
	LatestStatusChangeSlot uint64
	// Sequence number when validator exited (or 0)
	ExitCount uint64
	// LastPoCChangeSlot is the last time the PoC was changed
	LastPoCChangeSlot uint64
	// SecondLastPoCChangeSlot is the second to last time the PoC was changed
	SecondLastPoCChangeSlot uint64
}

// GetPublicKey gets the cached validator pubkey.
func (v *Validator) GetPublicKey() (*bls.PublicKey, error) {
	if v.XXXPubkeyCached == nil {
		pub, err := bls.DeserializePublicKey(v.Pubkey)
		if err != nil {
			return nil, err
		}
		v.XXXPubkeyCached = pub
	}
	return v.XXXPubkeyCached, nil
}

// Copy copies a validator instance.
func (v *Validator) Copy() Validator {
	return *v
}

// IsActive checks if the validator is active.
func (v Validator) IsActive() bool {
	return v.Status == Active || v.Status == ActivePendingExit
}

// ValidatorFromProto gets the validator for the protobuf representation
func ValidatorFromProto(validator *pb.Validator) (*Validator, error) {
	if len(validator.Pubkey) != 96 {
		return nil, errors.New("validator pubkey should be 96 bytes")
	}
	v := &Validator{
		Status:                  validator.Status,
		LatestStatusChangeSlot:  validator.LatestStatusChangeSlot,
		ExitCount:               validator.LatestStatusChangeSlot,
		LastPoCChangeSlot:       validator.LastPoCChangeSlot,
		SecondLastPoCChangeSlot: validator.SecondLastPoCChangeSlot,
	}
	err := v.WithdrawalCredentials.SetBytes(validator.WithdrawalCredentials)
	if err != nil {
		return nil, err
	}
	copy(v.Pubkey[:], validator.Pubkey)

	return v, nil
}

// GetActiveValidatorIndices gets validator indices that are active.
func GetActiveValidatorIndices(validators []Validator) []uint32 {
	var active []uint32
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
		Pubkey:                  v.Pubkey[:],
		WithdrawalCredentials:   v.WithdrawalCredentials[:],
		LastPoCChangeSlot:       v.LastPoCChangeSlot,
		SecondLastPoCChangeSlot: v.SecondLastPoCChangeSlot,
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

// ToProto gets the protobuf representation of the crosslink
func (c *Crosslink) ToProto() *pb.Crosslink {
	return &pb.Crosslink{
		Slot:           c.Slot,
		ShardBlockHash: c.ShardBlockHash[:],
	}
}

// CrosslinkFromProto gets the crosslink for a protobuf representation
func CrosslinkFromProto(crosslink *pb.Crosslink) (*Crosslink, error) {
	c := &Crosslink{Slot: crosslink.Slot}
	err := c.ShardBlockHash.SetBytes(crosslink.ShardBlockHash)
	if err != nil {
		return nil, err
	}
	return c, nil
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
	newSc.Committee = append([]uint32{}, sc.Committee...)
	return newSc
}

// ToProto gets the protobuf representation of the shard and committee
func (sc *ShardAndCommittee) ToProto() *pb.ShardCommittee {
	return &pb.ShardCommittee{
		Shard:               sc.Shard,
		Committee:           sc.Committee,
		TotalValidatorCount: sc.TotalValidatorCount,
	}
}

// ShardAndCommitteeFromProto gets the shard and committee for the protobuf representation.
func ShardAndCommitteeFromProto(committee *pb.ShardCommittee) (*ShardAndCommittee, error) {
	sc := &ShardAndCommittee{
		Shard:               committee.Shard,
		TotalValidatorCount: committee.TotalValidatorCount,
	}
	sc.Committee = append([]uint32{}, committee.Committee...)
	return sc, nil
}
