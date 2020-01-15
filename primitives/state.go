package primitives

import (
	"bytes"
	"fmt"

	"github.com/prysmaticlabs/go-ssz"

	"github.com/pkg/errors"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
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

// Crosslink goes in a collation to represent the last crystallized beacon block.
type Crosslink struct {
	// Slot is the slot within the current dynasty.
	Slot uint64

	// Shard chain block hash
	ShardBlockHash chainhash.Hash

	// Shard chain state hash
	ShardStateHash chainhash.Hash
}

// ToProto gets the protobuf representation of the crosslink
func (c *Crosslink) ToProto() *pb.Crosslink {
	return &pb.Crosslink{
		Slot:           c.Slot,
		ShardBlockHash: c.ShardBlockHash[:],
		ShardStateHash: c.ShardStateHash[:],
	}
}

// CrosslinkFromProto gets the crosslink for a protobuf representation
func CrosslinkFromProto(crosslink *pb.Crosslink) (*Crosslink, error) {
	c := &Crosslink{Slot: crosslink.Slot}
	err := c.ShardBlockHash.SetBytes(crosslink.ShardBlockHash)
	if err != nil {
		return nil, err
	}
	err = c.ShardStateHash.SetBytes(crosslink.ShardStateHash)
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

// State is the state of a beacon block
type State struct {
	// Slot is the current slot.
	Slot uint64

	// EpochIndex is the index of the last processed epoch.
	EpochIndex uint64

	// GenesisTime is the time of the genesis block.
	GenesisTime uint64

	// ForkData is the versioning data for forks.
	ForkData ForkData

	// ValidatorRegistry is the registry mapping IDs to validators. This stores
	// information about the validator's public key and statue.
	ValidatorRegistry []Validator

	// ValidatorBalances are the balances corresponding to each validator.
	ValidatorBalances []uint64

	// ValidatorRegistryLatestChangeSlot is the slot where the validator registry
	// last changed.
	ValidatorRegistryLatestChangeEpoch uint64

	// ValidatorRegistryExitCount is the number of validators that are exited.
	ValidatorRegistryExitCount uint64

	// ValidatorSetDeltaHashChange is for light clients to keep track of validator
	// registry changes.
	ValidatorRegistryDeltaChainTip chainhash.Hash

	// RandaoMix is the mix of randao reveals to provide entropy.
	RandaoMix chainhash.Hash

	// NextRandaoMix is the mix of randao reveals that is currently being generated.
	NextRandaoMix chainhash.Hash

	// COMMITTEES
	// ShardAndCommitteeForSlots is a list of committee members
	// and their assigned shard, per slot
	ShardAndCommitteeForSlots [][]ShardAndCommittee

	// FINALITY
	PreviousJustifiedEpoch uint64
	JustifiedEpoch         uint64
	JustificationBitfield  uint64
	FinalizedEpoch         uint64

	// RECENT STATE
	LatestCrosslinks          []Crosslink
	PreviousCrosslinks        []Crosslink
	LatestBlockHashes         []chainhash.Hash
	CurrentEpochAttestations  []PendingAttestation
	PreviousEpochAttestations []PendingAttestation
	BatchedBlockRoots         []chainhash.Hash

	ShardRegistry []chainhash.Hash
	Proposals     []ActiveProposal
	PendingVotes  []AggregatedVote
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
	newShardRegistry := make([]chainhash.Hash, len(s.ShardRegistry))
	for i := range s.ShardRegistry {
		copy(newShardRegistry[i][:], s.ShardRegistry[i][:])
	}
	newProposals := make([]ActiveProposal, len(s.Proposals))
	for i := range s.Proposals {
		newProposals[i] = s.Proposals[i].Copy()
	}
	newPendingVotes := make([]AggregatedVote, len(s.PendingVotes))
	for i := range s.PendingVotes {
		newPendingVotes[i] = s.PendingVotes[i].Copy()
	}
	var newValidatorRegistryDeltaChainTip chainhash.Hash
	var newRandaoMix chainhash.Hash
	var newNextRandaoMix chainhash.Hash
	copy(newValidatorRegistryDeltaChainTip[:], s.ValidatorRegistryDeltaChainTip[:])
	copy(newRandaoMix[:], s.RandaoMix[:])
	copy(newNextRandaoMix[:], s.NextRandaoMix[:])

	newShardAndCommitteeForSlots := make([][]ShardAndCommittee, len(s.ShardAndCommitteeForSlots))
	for i, slot := range s.ShardAndCommitteeForSlots {
		newShardAndCommitteeForSlots[i] = make([]ShardAndCommittee, len(slot))
		for n, committee := range slot {
			newShardAndCommitteeForSlots[i][n] = committee.Copy()
		}
	}

	newLatestCrosslinks := make([]Crosslink, len(s.LatestCrosslinks))
	copy(newLatestCrosslinks, s.LatestCrosslinks)

	newPreviousCrosslinks := make([]Crosslink, len(s.PreviousCrosslinks))
	copy(newPreviousCrosslinks, s.PreviousCrosslinks)

	newLatestBlockHashes := make([]chainhash.Hash, len(s.LatestBlockHashes))
	copy(newLatestBlockHashes, s.LatestBlockHashes)
	newCurrentAttestations := make([]PendingAttestation, len(s.CurrentEpochAttestations))
	for i := range s.CurrentEpochAttestations {
		newCurrentAttestations[i] = s.CurrentEpochAttestations[i].Copy()
	}

	newPreviousAttestations := make([]PendingAttestation, len(s.PreviousEpochAttestations))
	for i := range s.PreviousEpochAttestations {
		newPreviousAttestations[i] = s.PreviousEpochAttestations[i].Copy()
	}
	newBatchedBlockRoots := make([]chainhash.Hash, len(s.BatchedBlockRoots))
	copy(newBatchedBlockRoots, s.BatchedBlockRoots)
	newState := State{
		Slot:                               s.Slot,
		GenesisTime:                        s.GenesisTime,
		ForkData:                           s.ForkData.Copy(),
		EpochIndex:                         s.EpochIndex,
		ValidatorRegistry:                  newValidatorRegistry,
		ValidatorBalances:                  newValidatorBalances,
		ValidatorRegistryLatestChangeEpoch: s.ValidatorRegistryLatestChangeEpoch,
		ValidatorRegistryExitCount:         s.ValidatorRegistryExitCount,
		ValidatorRegistryDeltaChainTip:     newValidatorRegistryDeltaChainTip,
		RandaoMix:                          newRandaoMix,
		NextRandaoMix:                      newNextRandaoMix,
		ShardAndCommitteeForSlots:          newShardAndCommitteeForSlots,
		PreviousJustifiedEpoch:             s.PreviousJustifiedEpoch,
		JustifiedEpoch:                     s.JustifiedEpoch,
		JustificationBitfield:              s.JustificationBitfield,
		FinalizedEpoch:                     s.FinalizedEpoch,
		LatestCrosslinks:                   newLatestCrosslinks,
		PreviousCrosslinks:                 newPreviousCrosslinks,
		LatestBlockHashes:                  newLatestBlockHashes,
		PreviousEpochAttestations:          newPreviousAttestations,
		CurrentEpochAttestations:           newCurrentAttestations,
		BatchedBlockRoots:                  newBatchedBlockRoots,
		ShardRegistry:                      newShardRegistry,
		PendingVotes:                       newPendingVotes,
		Proposals:                          newProposals,
	}

	return newState
}

// ToProto gets the protobuf representation of the state.
func (s *State) ToProto() *pb.State {
	validatorRegistry := make([]*pb.Validator, len(s.ValidatorRegistry))
	shardCommittees := make([]*pb.ShardCommitteesForSlot, len(s.ShardAndCommitteeForSlots))
	latestCrosslinks := make([]*pb.Crosslink, len(s.LatestCrosslinks))
	previousCrosslinks := make([]*pb.Crosslink, len(s.PreviousCrosslinks))
	latestBlockHashes := make([][]byte, len(s.LatestBlockHashes))
	currentAttestations := make([]*pb.PendingAttestation, len(s.CurrentEpochAttestations))
	previousAttestations := make([]*pb.PendingAttestation, len(s.PreviousEpochAttestations))
	batchedBlockRoots := make([][]byte, len(s.BatchedBlockRoots))
	proposals := make([]*pb.ActiveProposal, len(s.Proposals))
	shardRegistry := make([][]byte, len(s.ShardRegistry))
	pendingVotes := make([]*pb.AggregatedVote, len(s.PendingVotes))

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

	for i := range previousCrosslinks {
		previousCrosslinks[i] = s.PreviousCrosslinks[i].ToProto()
	}

	for i := range latestBlockHashes {
		latestBlockHashes[i] = s.LatestBlockHashes[i][:]
	}

	for i := range currentAttestations {
		currentAttestations[i] = s.CurrentEpochAttestations[i].ToProto()
	}

	for i := range previousAttestations {
		previousAttestations[i] = s.PreviousEpochAttestations[i].ToProto()
	}

	for i := range proposals {
		proposals[i] = s.Proposals[i].ToProto()
	}

	for i := range pendingVotes {
		pendingVotes[i] = s.PendingVotes[i].ToProto()
	}

	for i := range shardRegistry {
		shardRegistry[i] = s.ShardRegistry[i][:]
	}

	for i := range batchedBlockRoots {
		batchedBlockRoots[i] = s.BatchedBlockRoots[i][:]
	}

	return &pb.State{
		Slot:                               s.Slot,
		GenesisTime:                        s.GenesisTime,
		EpochIndex:                         s.EpochIndex,
		ForkData:                           s.ForkData.ToProto(),
		ValidatorRegistry:                  validatorRegistry,
		ValidatorBalances:                  s.ValidatorBalances,
		ValidatorRegistryLatestChangeEpoch: s.ValidatorRegistryLatestChangeEpoch,
		ValidatorRegistryDeltaChainTip:     s.ValidatorRegistryDeltaChainTip[:],
		ValidatorRegistryExitCount:         s.ValidatorRegistryExitCount,
		RandaoMix:                          s.RandaoMix[:],
		NextRandaoMix:                      s.NextRandaoMix[:],
		ShardCommittees:                    shardCommittees,
		PreviousJustifiedEpoch:             s.PreviousJustifiedEpoch,
		JustifiedEpoch:                     s.JustifiedEpoch,
		JustificationBitField:              s.JustificationBitfield,
		FinalizedEpoch:                     s.FinalizedEpoch,
		LatestCrosslinks:                   latestCrosslinks,
		PreviousCrosslinks:                 previousCrosslinks,
		LatestBlockHashes:                  latestBlockHashes,
		CurrentEpochAttestations:           currentAttestations,
		PreviousEpochAttestations:          previousAttestations,
		BatchedBlockRoots:                  batchedBlockRoots,
		ShardRegistry:                      shardRegistry,
		PendingVotes:                       pendingVotes,
		Proposals:                          proposals,
	}
}

// StateFromProto gets the state fromo the protobuf representation.
func StateFromProto(s *pb.State) (*State, error) {
	validatorRegistry := make([]Validator, len(s.ValidatorRegistry))
	shardAndCommitteeForSlots := make([][]ShardAndCommittee, len(s.ShardCommittees))
	latestCrosslinks := make([]Crosslink, len(s.LatestCrosslinks))
	previousCrosslinks := make([]Crosslink, len(s.PreviousCrosslinks))
	latestBlockHashes := make([]chainhash.Hash, len(s.LatestBlockHashes))
	currentAttestations := make([]PendingAttestation, len(s.CurrentEpochAttestations))
	previousAttestations := make([]PendingAttestation, len(s.PreviousEpochAttestations))
	batchedBlockRoots := make([]chainhash.Hash, len(s.BatchedBlockRoots))
	pendingVotes := make([]AggregatedVote, len(s.PendingVotes))
	proposals := make([]ActiveProposal, len(s.Proposals))
	shardRegistry := make([]chainhash.Hash, len(s.ShardRegistry))

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

	for i := range previousCrosslinks {
		c, err := CrosslinkFromProto(s.PreviousCrosslinks[i])
		if err != nil {
			return nil, err
		}
		previousCrosslinks[i] = *c
	}

	for i := range latestBlockHashes {
		err := latestBlockHashes[i].SetBytes(s.LatestBlockHashes[i])
		if err != nil {
			return nil, err
		}
	}

	for i := range previousAttestations {
		a, err := PendingAttestationFromProto(s.PreviousEpochAttestations[i])
		if err != nil {
			return nil, err
		}
		previousAttestations[i] = *a
	}

	for i := range proposals {
		a, err := ActiveProposalFromProto(s.Proposals[i])
		if err != nil {
			return nil, err
		}
		proposals[i] = *a
	}

	for i := range pendingVotes {
		a, err := AggregatedVoteFromProto(s.PendingVotes[i])
		if err != nil {
			return nil, err
		}
		pendingVotes[i] = *a
	}

	for i := range currentAttestations {
		a, err := PendingAttestationFromProto(s.CurrentEpochAttestations[i])
		if err != nil {
			return nil, err
		}
		currentAttestations[i] = *a
	}

	for i := range batchedBlockRoots {
		err := batchedBlockRoots[i].SetBytes(s.BatchedBlockRoots[i])
		if err != nil {
			return nil, err
		}
	}

	for i := range shardRegistry {
		err := shardRegistry[i].SetBytes(s.ShardRegistry[i])
		if err != nil {
			return nil, err
		}
	}

	newState := &State{
		Slot:                               s.Slot,
		GenesisTime:                        s.GenesisTime,
		EpochIndex:                         s.EpochIndex,
		ForkData:                           *fd,
		ValidatorRegistry:                  validatorRegistry,
		ValidatorRegistryLatestChangeEpoch: s.ValidatorRegistryLatestChangeEpoch,
		ValidatorRegistryExitCount:         s.ValidatorRegistryExitCount,
		ShardAndCommitteeForSlots:          shardAndCommitteeForSlots,
		PreviousJustifiedEpoch:             s.PreviousJustifiedEpoch,
		JustifiedEpoch:                     s.JustifiedEpoch,
		JustificationBitfield:              s.JustificationBitField,
		FinalizedEpoch:                     s.FinalizedEpoch,
		LatestCrosslinks:                   latestCrosslinks,
		PreviousCrosslinks:                 previousCrosslinks,
		LatestBlockHashes:                  latestBlockHashes,
		CurrentEpochAttestations:           currentAttestations,
		PreviousEpochAttestations:          previousAttestations,
		BatchedBlockRoots:                  batchedBlockRoots,
		ShardRegistry:                      shardRegistry,
		PendingVotes:                       pendingVotes,
		Proposals:                          proposals,
	}

	err = newState.ValidatorRegistryDeltaChainTip.SetBytes(s.ValidatorRegistryDeltaChainTip)
	if err != nil {
		return nil, err
	}

	err = newState.RandaoMix.SetBytes(s.RandaoMix)
	if err != nil {
		return nil, err
	}

	err = newState.NextRandaoMix.SetBytes(s.NextRandaoMix)
	if err != nil {
		return nil, err
	}

	newState.ValidatorBalances = append([]uint64{}, s.ValidatorBalances...)

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
	return ssz.HashTreeRoot(ValidatorRegistryDeltaBlock{
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

// GetAssignmentsAssumingShuffle gets the assignments for the next epoch assuming the validators
// are shuffled.
func (s *State) GetAssignmentsAssumingShuffle(c *config.Config) [][]ShardAndCommittee {
	startShard := s.ShardAndCommitteeForSlots[0][0].Shard

	// epochsSinceLastRegistryChange is a power of 2
	return getNewShuffling(s.RandaoMix, s.ValidatorRegistry, int(startShard), c)
}

// GetShardProposerIndex gets the proposer index for a certain shard at a certain slot.
func (s *State) GetShardProposerIndex(slot uint64, shard uint64, c *config.Config) (uint32, error) {
	// find the committee pertaining to that shard
	var shardCommittee []uint32
	for _, slotCommittees := range s.ShardAndCommitteeForSlots {
		for _, committee := range slotCommittees {
			if committee.Shard == shard {
				shardCommittee = committee.Committee
			}
		}
	}
	if shardCommittee == nil {
		return 0, fmt.Errorf("no shard committee for shard %d", shard)
	}
	// offset by 1 so it's not the same as the beacon proposer
	return shardCommittee[int(slot+1)%len(shardCommittee)], nil
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
		whistleblowerIndex, err := s.GetBeaconProposerIndex(s.Slot-1, c)
		if err != nil {
			return err
		}
		whistleblowerReward := s.GetEffectiveBalance(index, c) / c.WhistleblowerRewardQuotient

		s.ValidatorBalances[whistleblowerIndex] += whistleblowerReward
		s.ValidatorBalances[index] -= whistleblowerReward
	}

	s.ValidatorRegistryExitCount++
	validator.ExitCount = s.ValidatorRegistryExitCount
	deltaChainTip, err := GetNewValidatorRegistryDeltaChainTip(s.ValidatorRegistryDeltaChainTip, index, validator.Pubkey, ExitFlag)
	if err != nil {
		return err
	}
	s.ValidatorRegistryDeltaChainTip = deltaChainTip

	for i := range s.Proposals {
		participation := s.Proposals[i].Participation

		if participation[index/8]&(1<<uint(index%8)) != 0 {
			// switch the participation bit off
			s.Proposals[i].Participation[index/8] &= ^(1 << uint(index%8))
		}
	}
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

	h, err := ssz.HashTreeRoot(pubkey.Serialize())
	if err != nil {
		return false, err
	}
	valid, err := bls.VerifySig(pubkey, h[:], &proofOfPossession, bls.DomainDeposit)
	if err != nil {
		return false, err
	}
	return valid, nil
}

// applyProposerSlashing applies a proposer slashing if valid.
func (s *State) applyProposerSlashing(proposerSlashing ProposerSlashing, config *config.Config) error {
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
	hashProposal1, err := ssz.HashTreeRoot(proposerSlashing.ProposalData1)
	if err != nil {
		return err
	}
	hashProposal2, err := ssz.HashTreeRoot(proposerSlashing.ProposalData2)
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

func isDoubleVote(ad1 AttestationData, ad2 AttestationData) bool {
	targetEpoch1 := ad1.TargetEpoch
	targetEpoch2 := ad2.TargetEpoch
	return targetEpoch1 == targetEpoch2
}

func isSurroundVote(ad1 AttestationData, ad2 AttestationData) bool {
	targetEpoch1 := ad1.TargetEpoch
	targetEpoch2 := ad2.TargetEpoch
	sourceEpoch1 := ad1.SourceEpoch
	sourceEpoch2 := ad2.SourceEpoch
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

	ad0Hash, err := ssz.HashTreeRoot(ad0)
	if err != nil {
		return false
	}

	ad1Hash, err := ssz.HashTreeRoot(ad1)
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

// applyCasperSlashing applies a casper slashing claim to the current state.
func (s *State) applyCasperSlashing(casperSlashing CasperSlashing, c *config.Config) error {
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

	if !isDoubleVote(casperSlashing.Votes1.Data, casperSlashing.Votes2.Data) &&
		!isSurroundVote(casperSlashing.Votes1.Data, casperSlashing.Votes2.Data) {
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
	shardCommittees, err := s.GetShardCommitteesAtSlot(data.Slot-1, c)
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
	// increase the slot number
	s.Slot++

	s.LatestBlockHashes[(s.Slot-1)%c.LatestBlockRootsLength] = previousBlockRoot

	if s.Slot%c.LatestBlockRootsLength == 0 {
		latestBlockHashesRoot, err := ssz.HashTreeRoot(s.LatestBlockHashes)
		if err != nil {
			return err
		}
		s.BatchedBlockRoots = append(s.BatchedBlockRoots, latestBlockHashesRoot)
	}

	return nil
}

// BlockView is an interface the provides access to blocks.
type BlockView interface {
	GetHashBySlot(slot uint64) (chainhash.Hash, error)
	Tip() (chainhash.Hash, error)
	SetTipSlot(slot uint64)
	GetLastStateRoot() (chainhash.Hash, error)
}

// ProcessSlots uses the current head to process slots up to a certain slot, applying
// slot transitions and epoch transitions, and returns the updated state. Note that this should
// only process up to the current slot number so that the lastBlockHash remains constant.
func (s *State) ProcessSlots(upTo uint64, view BlockView, c *config.Config) ([]Receipt, error) {
	var receipts []Receipt

	//initialSlot := s.Slot

	for s.Slot < upTo {
		// this only happens when there wasn't a block at the first slot of the epoch
		if s.Slot/c.EpochLength > s.EpochIndex && s.Slot%c.EpochLength == 0 {
			epochReceipts, err := s.ProcessEpochTransition(c)
			if err != nil {
				return nil, err
			}

			//fmt.Printf("ProcessSlots...,   ProcessEpochTransition lenOfEpochReceipts=%d s.Slot=%d c.EpochLength=%d\n", len(epochReceipts), s.Slot, c.EpochLength)
			receipts = append(receipts, epochReceipts...)
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

	/* stats code for test the memory usage, will remove it after we fix the memory usage.
	statsMap := map[uint32]uint32{}
	for _, r := range receipts {
		index := r.Index
		_, found := statsMap[index]
		if found {
			statsMap[index]++
		} else {
			statsMap[index] = 1
		}
	}
	validatorCount := uint32(0)
	maxDuplicatedCount := uint32(0)
	minDuplicatedCount := uint32(99999999)
	for _, v := range statsMap {
		validatorCount++
		if v > maxDuplicatedCount {
			maxDuplicatedCount = v
		}
		if v < minDuplicatedCount {
			minDuplicatedCount = v
		}
	}

	//fmt.Printf("ProcessSlots s.Slot=%d upTo=%d lenOfReceipts=%d validatorCount=%d maxDuplicatedCount=%d minDuplicatedCount=%d, below is the stack trace\n",
	//	initialSlot, upTo, len(receipts), validatorCount, maxDuplicatedCount, minDuplicatedCount)
	//debug.PrintStack()
	*/

	return receipts, nil
}
