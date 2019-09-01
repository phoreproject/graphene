package primitives

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/prysmaticlabs/go-ssz"
	"math"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/sirupsen/logrus"
)

func intSqrt(n uint64) uint64 {
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// shouldUpdateRegistry returns if the validator registry is ready to be shuffled.
// The registry should only be updated if every shard has created a crosslink since the
// last update and the beacon chain has been finalized since the last update.
func (s *State) shouldUpdateRegistry() bool {
	if s.FinalizedEpoch <= s.ValidatorRegistryLatestChangeEpoch {
		return false
	}

	for _, shardAndCommittees := range s.ShardAndCommitteeForSlots {
		for _, committee := range shardAndCommittees {
			if s.LatestCrosslinks[committee.Shard].Slot <= s.ValidatorRegistryLatestChangeEpoch {
				return false
			}
		}
	}

	return true
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
func getNewShuffling(seed chainhash.Hash, validators []Validator, crosslinkingStart int, con *config.Config) [][]ShardAndCommittee {
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
	case AttestedInPreviousEpoch:
		return "attested to target matching previous epoch"
	case AttestedToMatchingTargetHash:
		return "attested to correct target hash"
	case AttestedToCorrectBeaconBlockHash:
		return "attested to correct beacon block"
	case AttestationInclusionDistanceReward:
		return "attestation inclusion distance reward"
	case AttestedToIncorrectBeaconBlockHash:
		return "did not attest to correct beacon block"
	case AttestedToIncorrectTargetHash:
		return "did not attest to correct target hash"
	case DidNotAttestToPreviousEpoch:
		return "did not attest to target matching previous epoch"
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
	// AttestedInPreviousEpoch is a reward when a validator attests to the previous epoch justified slot.
	AttestedInPreviousEpoch = iota

	// AttestedToMatchingTargetHash is a reward when a validator attests to the previous epoch boundary hash.
	AttestedToMatchingTargetHash

	// AttestedToCorrectBeaconBlockHash is a reward when a validator attests to the correct beacon block hash for their slot.
	AttestedToCorrectBeaconBlockHash

	// AttestationInclusionDistanceReward is a reward for including attestations in beacon blocks.
	AttestationInclusionDistanceReward

	// AttestedToIncorrectBeaconBlockHash is a penalty for not attesting to the correct beacon block.
	AttestedToIncorrectBeaconBlockHash

	// AttestedToIncorrectTargetHash is a penalty for not attesting to the previous epoch boundary.
	AttestedToIncorrectTargetHash

	// DidNotAttestToPreviousEpoch is a penalty for not attesting to the previous justified slot.
	DidNotAttestToPreviousEpoch

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

// GetRecentBlockHash gets block hashes from the LatestBlockHashes array.
func (s *State) GetRecentBlockHash(slotToGet uint64, c *config.Config) (*chainhash.Hash, error) {
	if s.Slot-slotToGet >= c.LatestBlockRootsLength {
		return nil, errors.New("can't fetch block hash from more than LatestBlockRootsLength")
	}

	return &s.LatestBlockHashes[slotToGet%c.LatestBlockRootsLength], nil
}

// ProcessEpochTransition processes an epoch transition and modifies state. This shouldn't usually
// be used as ProcessSlots is generally a better way to update state, but sometimes it's required to
// validate/generate attestations for the next epoch.
func (s *State) ProcessEpochTransition(c *config.Config) ([]Receipt, error) {
	activeValidatorIndices := GetActiveValidatorIndices(s.ValidatorRegistry)
	totalBalance := s.GetTotalBalance(activeValidatorIndices, c)

	// s.PreviousEpochAttestations are all of the attestations that use the previous epoch as the source
	// s.CurrentEpochAttestations are all of the attestations that use the current epoch as the source

	// previousEpochAttesterIndices are all participants of attestations in the previous epoch
	previousEpochAttesterIndices := map[uint32]struct{}{}
	for _, a := range s.PreviousEpochAttestations {
		participants, err := s.GetAttestationParticipants(a.Data, a.ParticipationBitfield, c)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			previousEpochAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochAttestingBalance := s.GetTotalBalanceMap(previousEpochAttesterIndices, c)

	previousEpochBoundaryHash := chainhash.Hash{}
	if s.Slot >= 2*c.EpochLength {
		ebhm2, err := s.GetRecentBlockHash(s.Slot-2*c.EpochLength, c)
		if err != nil {
			ebhm2 = &chainhash.Hash{}
		}
		previousEpochBoundaryHash = *ebhm2
	}

	currentEpochBoundaryHash := chainhash.Hash{}
	if s.Slot >= c.EpochLength {
		ebhm1, err := s.GetRecentBlockHash(s.Slot-c.EpochLength, c)
		if err != nil {
			ebhm1 = &chainhash.Hash{}
		}
		currentEpochBoundaryHash = *ebhm1
	}

	// attestationsMatchingPreviousTarget is any attestation where the source is the previous epoch and the
	// target is the previousEpochBoundaryHash
	var attestationsMatchingPreviousTarget []PendingAttestation
	for _, a := range s.PreviousEpochAttestations {
		if previousEpochBoundaryHash.IsEqual(&a.Data.TargetHash) {
			attestationsMatchingPreviousTarget = append(attestationsMatchingPreviousTarget, a)
		}
	}

	// attestationsMatchingCurrentTarget is any attestation in the previous epoch where the epoch boundary is
	// set to the epoch boundary two epochs ago
	var attestationsMatchingCurrentTarget []PendingAttestation
	for _, a := range s.CurrentEpochAttestations {
		// source is current epoch and
		if currentEpochBoundaryHash.IsEqual(&a.Data.TargetHash) {
			attestationsMatchingCurrentTarget = append(attestationsMatchingCurrentTarget, a)
		}
	}

	attestersMatchingPreviousTarget := map[uint32]struct{}{}
	for _, a := range attestationsMatchingPreviousTarget {
		participants, err := s.GetAttestationParticipants(a.Data, a.ParticipationBitfield, c)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			attestersMatchingPreviousTarget[p] = struct{}{}
		}
	}

	attestersMatchingCurrentTarget := map[uint32]struct{}{}
	for _, a := range attestationsMatchingCurrentTarget {
		participants, err := s.GetAttestationParticipants(a.Data, a.ParticipationBitfield, c)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			attestersMatchingCurrentTarget[p] = struct{}{}
		}
	}

	previousEpochBoundaryAttestingBalance := s.GetTotalBalanceMap(attestersMatchingPreviousTarget, c)
	currentEpochBoundaryAttestingBalance := s.GetTotalBalanceMap(attestersMatchingCurrentTarget, c)

	var previousEpochAttestationsMatchingBeaconBlock []PendingAttestation
	for _, a := range s.PreviousEpochAttestations {
		blockRoot, err := s.GetRecentBlockHash(a.Data.Slot, c)
		if err != nil {
			break
		}
		if a.Data.BeaconBlockHash.IsEqual(blockRoot) {
			previousEpochAttestationsMatchingBeaconBlock = append(previousEpochAttestationsMatchingBeaconBlock, a)
		}
	}

	previousEpochHeadAttesterIndices := map[uint32]struct{}{}
	for _, a := range s.PreviousEpochAttestations {
		participants, err := s.GetAttestationParticipants(a.Data, a.ParticipationBitfield, c)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			previousEpochHeadAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochHeadAttestingBalance := s.GetTotalBalanceMap(previousEpochHeadAttesterIndices, c)

	oldPreviousJustifiedEpoch := s.PreviousJustifiedEpoch

	s.PreviousJustifiedEpoch = s.JustifiedEpoch
	s.PreviousCrosslinks = s.LatestCrosslinks[:]
	s.JustificationBitfield = s.JustificationBitfield * 2

	if 3*previousEpochBoundaryAttestingBalance >= 2*totalBalance {
		s.JustificationBitfield |= 1 << 1 // mark the last epoch as justified
		s.JustifiedEpoch = s.EpochIndex - 1
	}
	if 3*currentEpochBoundaryAttestingBalance >= 2*totalBalance {
		s.JustificationBitfield |= 1 << 0 // mark the last epoch as justified
		s.JustifiedEpoch = s.EpochIndex
	}

	if ((s.JustificationBitfield>>1)%8 == 7 && s.PreviousJustifiedEpoch == s.EpochIndex-2) || // 1110
		((s.JustificationBitfield>>1)%4 == 3 && s.PreviousJustifiedEpoch == s.EpochIndex-1) { // 110 <- old previous justified would be
		s.FinalizedEpoch = oldPreviousJustifiedEpoch
	}

	if ((s.JustificationBitfield>>0)%8 == 7 && s.JustifiedEpoch == s.EpochIndex-1) ||
		((s.JustificationBitfield>>0)%4 == 3 && s.JustifiedEpoch == s.EpochIndex) {
		s.FinalizedEpoch = s.PreviousJustifiedEpoch
	}

	// attestingValidatorIndices gets the participants that attested to a certain shardblockRoot for a certain shardCommittee
	attestingValidatorIndices := func(shardComittee ShardAndCommittee, shardBlockRoot chainhash.Hash) ([]uint32, error) {
		outMap := map[uint32]struct{}{}
		for _, a := range s.CurrentEpochAttestations {
			if a.Data.Shard == shardComittee.Shard && a.Data.ShardBlockHash.IsEqual(&shardBlockRoot) {
				for i, s := range shardComittee.Committee {
					bit := a.ParticipationBitfield[i/8] & (1 << uint(i%8))
					if bit != 0 {
						outMap[s] = struct{}{}
					}
				}
			}
		}
		for _, a := range s.PreviousEpochAttestations {
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
		for _, a := range s.CurrentEpochAttestations {
			if a.Data.Shard != shardCommittee.Shard {
				continue
			}
			possibleHashes[a.Data.ShardBlockHash] = struct{}{}
		}
		for _, a := range s.PreviousEpochAttestations {
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

			fmt.Println(bestRoot)

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

	epochsSinceFinality := s.EpochIndex - s.FinalizedEpoch

	// previousAttestationCache tracks which validator attested to which attestation
	previousAttestationCache := map[uint32]*PendingAttestation{}
	for _, a := range s.PreviousEpochAttestations {
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
					Type:   AttestedToCorrectBeaconBlockHash,
					Index:  index,
					Amount: int64(reward),
				})
			} else {
				penalty := baseReward(index)
				totalPenalized += penalty
				s.ValidatorBalances[index] -= penalty
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   AttestedToIncorrectBeaconBlockHash,
					Index:  index,
					Amount: -int64(penalty),
				})
			}

			// reward or slash based on if they attested to the correct source
			if _, found := attestersMatchingPreviousTarget[index]; found {
				reward := baseReward(index) * previousEpochBoundaryAttestingBalance / totalBalance
				totalRewarded += reward
				s.ValidatorBalances[index] += reward
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   AttestedToMatchingTargetHash,
					Index:  index,
					Amount: int64(reward),
				})
			} else {
				penalty := baseReward(index)
				totalPenalized += baseReward(index)
				s.ValidatorBalances[index] -= baseReward(index)
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   AttestedToIncorrectTargetHash,
					Index:  index,
					Amount: -int64(penalty),
				})
			}

			// reward or slash based on if they attested to the correct target
			if _, found := previousEpochAttesterIndices[index]; found {
				reward := baseReward(index) * previousEpochAttestingBalance / totalBalance
				totalRewarded += reward
				s.ValidatorBalances[index] += reward
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   AttestedInPreviousEpoch,
					Index:  index,
					Amount: int64(reward),
				})
			} else {
				penalty := baseReward(index)
				totalPenalized += penalty
				s.ValidatorBalances[index] -= penalty
				receipts = append(receipts, Receipt{
					Slot:   s.Slot,
					Type:   DidNotAttestToPreviousEpoch,
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

	// process proposals if needed
	if s.EpochIndex%c.EpochsPerVotingPeriod == 0 {
		toRemove := make(map[int]struct{})

		for idx := 0; idx < len(s.Proposals); idx++ {
			if _, found := toRemove[idx]; found {
				continue
			}

			activeProposal := s.Proposals[idx]

			if activeProposal.Queued {
				epochsSinceStart := s.EpochIndex - activeProposal.StartEpoch

				// if the vote passed after the grace period
				if epochsSinceStart/c.EpochsPerVotingPeriod > c.GracePeriod {
					for _, shard := range activeProposal.Data.Shards {
						s.ShardRegistry[shard] = activeProposal.Data.ActionHash
					}

					proposalHash, _ := ssz.HashTreeRoot(activeProposal.Data)

					for i := range s.Proposals {
						if s.Proposals[i].Data.Type == Cancel && bytes.Equal(s.Proposals[i].Data.ActionHash[:], proposalHash[:]) {
							// queue any cancellations for removal

							toRemove[i] = struct{}{}
						}
					}

					toRemove[idx] = struct{}{}

					continue
				}
			} else {
				yesVotes := uint64(0)
				total := uint64(len(s.ValidatorRegistry))

				// if a validator leaves after voting and the validator registry shrinks, their
				// vote is ignored
				for i := uint64(0); i < total; i++ {
					if activeProposal.Participation[i/8]&(1<<uint(i%8)) != 0 {
						yesVotes++
					}
				}

				// proposal passes -> queue it
				if activeProposal.Data.Type == Propose && c.QueueThresholdDenominator*yesVotes >= c.QueueThresholdNumerator*total {
					s.Proposals[idx].Queued = true

					for i := range s.Proposals {
						proposalHash, _ := ssz.HashTreeRoot(s.Proposals[i].Data)

						if bytes.Equal(activeProposal.Data.ActionHash[:], proposalHash[:]) {
							// queue any cancellations for removal

							toRemove[i] = struct{}{}
						}
					}

					continue
				}

				// cancel vote passes -> remove active proposal
				if activeProposal.Data.Type == Cancel && c.CancelThresholdDenominator*yesVotes >= c.CancelThresholdNumerator*total {
					for i := range s.Proposals {
						proposalHash, _ := ssz.HashTreeRoot(s.Proposals[i].Data)

						if s.Proposals[i].Queued == true && bytes.Equal(activeProposal.Data.ActionHash[:], proposalHash[:]) {
							// queue any cancellations for removal

							toRemove[i] = struct{}{}
						}
					}

					toRemove[idx] = struct{}{}
					continue
				}

				epochsSinceStart := s.EpochIndex - activeProposal.StartEpoch

				// vote fails -> remove it
				if epochsSinceStart/c.EpochsPerVotingPeriod > c.VotingTimeout && c.FailThresholdDenominator*yesVotes < c.FailThresholdNumerator*total {
					toRemove[idx] = struct{}{}
				}

				if epochsSinceStart/c.EpochsPerVotingPeriod > c.VotingExpiration {
					toRemove[idx] = struct{}{}
				}
			}
		}

		out := 0

		for i, p := range s.Proposals {
			if _, found := toRemove[i]; !found {
				s.Proposals[out] = p
				out++
			}
		}

		s.Proposals = s.Proposals[:out]
	}

	if err := s.exitValidatorsUnderMinimum(c); err != nil {
		return nil, err
	}

	shouldUpdateRegistry := s.shouldUpdateRegistry()

	s.EpochIndex = s.Slot / c.EpochLength

	// update registry if:
	// - a slot has been finalized after the last time the registry was changed
	// - every shard
	if shouldUpdateRegistry {
		err := s.UpdateValidatorRegistry(c)
		if err != nil {
			return nil, err
		}

		s.ValidatorRegistryLatestChangeEpoch = s.EpochIndex
		copy(s.ShardAndCommitteeForSlots[:c.EpochLength], s.ShardAndCommitteeForSlots[c.EpochLength:])
		lastSlot := s.ShardAndCommitteeForSlots[len(s.ShardAndCommitteeForSlots)-1]
		lastCommittee := lastSlot[len(lastSlot)-1]
		nextStartShard := (lastCommittee.Shard + 1) % uint64(c.ShardCount)
		newShuffling := getNewShuffling(s.RandaoMix, s.ValidatorRegistry, int(nextStartShard), c)
		copy(s.ShardAndCommitteeForSlots[c.EpochLength:], newShuffling)
	} else {
		copy(s.ShardAndCommitteeForSlots[:c.EpochLength], s.ShardAndCommitteeForSlots[c.EpochLength:])
		epochsSinceLastRegistryChange := s.EpochIndex - s.ValidatorRegistryLatestChangeEpoch
		startShard := s.ShardAndCommitteeForSlots[0][0].Shard

		// epochsSinceLastRegistryChange is a power of 2
		if epochsSinceLastRegistryChange&(epochsSinceLastRegistryChange-1) == 0 {
			newShuffling := getNewShuffling(s.RandaoMix, s.ValidatorRegistry, int(startShard), c)
			copy(s.ShardAndCommitteeForSlots[c.EpochLength:], newShuffling)
		}
	}

	copy(s.RandaoMix[:], s.NextRandaoMix[:])

	s.PreviousEpochAttestations = s.CurrentEpochAttestations

	s.CurrentEpochAttestations = make([]PendingAttestation, 0)

	return receipts, nil
}
