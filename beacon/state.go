package beacon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/primitives"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	logger "github.com/sirupsen/logrus"
)

// InitializeState initializes state to the genesis state according to the config.
func (b *Blockchain) InitializeState(initialValidators []InitialValidatorEntry, genesisTime uint64) error {
	b.stateLock.Lock()
	crosslinks := make([]primitives.Crosslink, b.config.ShardCount)

	for i := 0; i < b.config.ShardCount; i++ {
		crosslinks[i] = primitives.Crosslink{
			Slot:           b.config.InitialSlotNumber,
			ShardBlockHash: zeroHash,
		}
	}

	recentBlockHashes := make([]chainhash.Hash, b.config.LatestBlockRootsLength)
	for i := uint64(0); i < b.config.LatestBlockRootsLength; i++ {
		recentBlockHashes[i] = zeroHash
	}

	b.state = primitives.State{
		Slot:        0,
		GenesisTime: genesisTime,
		ForkData: primitives.ForkData{
			PreForkVersion:  b.config.InitialForkVersion,
			PostForkVersion: b.config.InitialForkVersion,
			ForkSlotNumber:  b.config.InitialSlotNumber,
		},
		ValidatorRegistry:                 []primitives.Validator{},
		ValidatorBalances:                 []uint64{},
		ValidatorRegistryLatestChangeSlot: b.config.InitialSlotNumber,
		ValidatorRegistryExitCount:        0,
		ValidatorRegistryDeltaChainTip:    chainhash.Hash{},

		RandaoMix:                 chainhash.Hash{},
		NextSeed:                  chainhash.Hash{},
		ShardAndCommitteeForSlots: [][]primitives.ShardAndCommittee{},

		PreviousJustifiedSlot: b.config.InitialSlotNumber,
		JustifiedSlot:         b.config.InitialSlotNumber,
		JustificationBitfield: 0,
		FinalizedSlot:         b.config.InitialSlotNumber,

		LatestCrosslinks:            crosslinks,
		LatestBlockHashes:           recentBlockHashes,
		LatestPenalizedExitBalances: []uint64{},
		LatestAttestations:          []primitives.PendingAttestation{},
		BatchedBlockRoots:           []chainhash.Hash{},
	}

	for _, deposit := range initialValidators {
		validatorIndex, err := b.state.ProcessDeposit(deposit.PubKey, deposit.DepositSize, deposit.ProofOfPossession, deposit.WithdrawalCredentials, deposit.RandaoCommitment, deposit.PoCCommitment, b.config)
		if err != nil {
			return err
		}

		if b.state.GetEffectiveBalance(validatorIndex, b.config) == b.config.MaxDeposit {
			err := b.state.UpdateValidatorStatus(validatorIndex, primitives.Active, b.config)
			if err != nil {
				return err
			}
		}
	}

	initialShuffling := GetNewShuffling(zeroHash, b.state.ValidatorRegistry, 0, b.config)
	b.state.ShardAndCommitteeForSlots = append(initialShuffling, initialShuffling...)

	sig := bls.EmptySignature.Copy()
	stateRoot, err := b.state.TreeHashSSZ()
	if err != nil {
		return err
	}

	block0 := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   0,
			StateRoot:    stateRoot,
			ParentRoot:   zeroHash,
			RandaoReveal: zeroHash,
			Signature:    *sig,
		},
		BlockBody: primitives.BlockBody{
			ProposerSlashings: []primitives.ProposerSlashing{},
			CasperSlashings:   []primitives.CasperSlashing{},
			Attestations:      []primitives.Attestation{},
			Deposits:          []primitives.Deposit{},
			Exits:             []primitives.Exit{},
		},
	}

	b.stateLock.Unlock()

	err = b.AddBlock(&block0)
	if err != nil {
		return err
	}

	return nil
}

// GetActiveValidatorIndices gets the indices of active validators.
func GetActiveValidatorIndices(vs []primitives.Validator) []uint32 {
	var l []uint32
	for i, v := range vs {
		if v.Status == primitives.Active {
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
func GetNewShuffling(seed chainhash.Hash, validators []primitives.Validator, crosslinkingStart int, con *config.Config) [][]primitives.ShardAndCommittee {
	activeValidators := GetActiveValidatorIndices(validators)
	numActiveValidators := len(activeValidators)

	// clamp between 1 and b.config.ShardCount / b.config.EpochLength
	committeesPerSlot := clamp(1, con.ShardCount/int(con.EpochLength), numActiveValidators/int(con.EpochLength)/con.TargetCommitteeSize)

	output := make([][]primitives.ShardAndCommittee, con.EpochLength)

	shuffledValidatorIndices := ShuffleValidators(activeValidators, seed)

	validatorsPerSlot := Split(shuffledValidatorIndices, uint32(con.EpochLength))

	for slot, slotIndices := range validatorsPerSlot {
		shardIndices := Split(slotIndices, uint32(committeesPerSlot))

		shardIDStart := crosslinkingStart + slot*committeesPerSlot

		shardCommittees := make([]primitives.ShardAndCommittee, len(shardIndices))
		for shardPosition, indices := range shardIndices {
			shardCommittees[shardPosition] = primitives.ShardAndCommittee{
				Shard:     uint64((shardIDStart + shardPosition) % con.ShardCount),
				Committee: indices,
			}
		}

		output[slot] = shardCommittees
	}
	return output
}

// checkTrailingZeros ensures there are a certain number of trailing
// zeros in provided byte.
func checkTrailingZeros(a byte, numZeros uint8) bool {
	i := uint8(a)
	if (i%128)/64 == 1 && numZeros == 7 {
		return false
	}
	if (i%64)/32 == 1 && numZeros >= 6 {
		return false
	}
	if (i%32)/16 == 1 && numZeros >= 5 {
		return false
	}
	if (i%16)/8 == 1 && numZeros >= 4 {
		return false
	}
	if (i%8)/4 == 1 && numZeros >= 3 {
		return false
	}
	if (i%4)/2 == 1 && numZeros >= 2 {
		return false
	}
	if (i%2) == 1 && numZeros >= 1 {
		return false
	}
	return true
}

func validateAttestationSlot(attestation *primitives.Attestation, parentBlock *primitives.Block, c *config.Config) error {
	if attestation.Data.Slot > parentBlock.SlotNumber {
		return errors.New("attestation slot number too high")
	}

	// verify attestation slot >= max(parent.slot - CYCLE_LENGTH + 1, 0)
	if attestation.Data.Slot < uint64(math.Max(float64(int64(parentBlock.SlotNumber)-int64(c.EpochLength)+1), 0)) {
		return errors.New("attestation slot number too low")
	}

	return nil
}

func (b *Blockchain) findAttestationPublicKey(attestation *primitives.Attestation, parentBlock *primitives.Block, c *config.Config) (*bls.PublicKey, error) {
	attestationIndicesForShards := b.state.GetShardCommitteesAtSlot(attestation.Data.Slot, b.config)
	var attestationIndices primitives.ShardAndCommittee
	found := false
	for _, s := range attestationIndicesForShards {
		if s.Shard == attestation.Data.Shard {
			attestationIndices = s
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("could not find shard id %d", attestation.Data.Shard)
	}

	if len(attestation.ParticipationBitfield) != (len(attestationIndices.Committee)+7)/8 {
		return nil, fmt.Errorf("attestation bitfield length does not match number of validators in committee")
	}

	trailingZeros := 8 - uint8(len(attestationIndices.Committee)%8)
	if trailingZeros != 8 && !checkTrailingZeros(attestation.ParticipationBitfield[len(attestation.ParticipationBitfield)-1], trailingZeros) {
		return nil, fmt.Errorf("expected %d bits at the end empty", trailingZeros)
	}

	pubkey := bls.NewAggregatePublicKey()
	for bit, n := range attestationIndices.Committee {
		set := (attestation.ParticipationBitfield[bit/8]>>uint(7-(bit%8)))%2 == 1
		if set {
			pubkey.AggregatePubKey(&b.state.ValidatorRegistry[n].Pubkey)
		}
	}

	return pubkey, nil
}

// AddBlock adds a block header to the current chain. The block should already
// have been validated by this point.
func (b *Blockchain) AddBlock(block *primitives.Block) error {
	blockHash, err := block.TreeHashSSZ()
	if err != nil {
		return err
	}
	logger.WithField("hash", blockHash).Debug("adding block to cache and updating head if needed")
	err = b.UpdateChainHead(block)
	if err != nil {
		return err
	}

	err = b.db.SetBlock(*block)
	if err != nil {
		return err
	}

	return nil
}

func repeatHash(data []byte, n uint64) []byte {
	if n == 0 {
		return data
	}
	return repeatHash(chainhash.HashB(data), n-1)
}

// ApplyBlock tries to apply a block to the state.
func (b *Blockchain) ApplyBlock(block *primitives.Block) error {
	// copy the state so we can easily revert
	newState := b.state.Copy()

	// increase the slot number
	newState.Slot++

	// increase the randao skips of the proposer
	newState.ValidatorRegistry[newState.GetBeaconProposerIndex(newState.Slot, b.config)].RandaoSkips++

	previousBlockRoot, err := b.GetNodeByHeight(block.SlotNumber - 1)
	if err != nil {
		return err
	}

	newState.LatestBlockHashes[(newState.Slot-1)%b.config.LatestBlockRootsLength] = previousBlockRoot

	if newState.Slot%b.config.LatestBlockRootsLength == 0 {
		latestBlockHashesRoot, err := ssz.TreeHash(newState.LatestBlockHashes)
		if err != nil {
			return err
		}
		newState.BatchedBlockRoots = append(newState.BatchedBlockRoots, latestBlockHashesRoot)
	}

	if block.SlotNumber != newState.Slot {
		return errors.New("block has incorrect slot number")
	}

	blockWithoutSignature := block.Copy()
	blockWithoutSignature.Signature = *bls.EmptySignature
	blockWithoutSignatureRoot, err := blockWithoutSignature.TreeHashSSZ()
	if err != nil {
		return err
	}

	proposal := primitives.ProposalSignedData{
		Slot:      newState.Slot,
		Shard:     b.config.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot, err := ssz.TreeHash(proposal)
	if err != nil {
		return err
	}

	beaconProposerIndex := newState.GetBeaconProposerIndex(newState.Slot, b.config)
	valid, err := bls.VerifySig(&newState.ValidatorRegistry[beaconProposerIndex].Pubkey, proposalRoot[:], &block.Signature, bls.DomainProposal)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("block had invalid signature")
	}

	proposer := &newState.ValidatorRegistry[beaconProposerIndex]
	expectedRandaoCommitment := repeatHash(block.RandaoReveal[:], proposer.RandaoSkips)

	if !bytes.Equal(expectedRandaoCommitment, proposer.RandaoCommitment[:]) {
		return errors.New("randao commitment does not match")
	}

	for i := range newState.RandaoMix {
		newState.RandaoMix[i] ^= block.RandaoReveal[i]
	}

	proposer.RandaoCommitment = block.RandaoReveal
	proposer.RandaoSkips = 0

	if len(block.BlockBody.ProposerSlashings) > b.config.MaxProposerSlashings {
		return errors.New("more than maximum proposer slashings")
	}

	if len(block.BlockBody.CasperSlashings) > b.config.MaxCasperSlashings {
		return errors.New("more than maximum casper slashings")
	}

	if len(block.BlockBody.Attestations) > b.config.MaxAttestations {
		return errors.New("more than maximum casper slashings")
	}

	for _, s := range block.BlockBody.ProposerSlashings {
		newState.ApplyProposerSlashing(s, b.config)
	}

	for _, c := range block.BlockBody.CasperSlashings {
		newState.ApplyCasperSlashing(c, b.config)
	}

	for _, a := range block.BlockBody.Attestations {
		b.ApplyAttestation(&newState, a, b.config)
	}

	// process deposits here

	for _, e := range block.BlockBody.Exits {
		newState.ApplyExit(e, b.config)
	}

	if newState.Slot%b.config.EpochLength == 0 {
		err := b.processEpochTransition(&newState)
		if err != nil {
			return err
		}
	}

	b.state = newState
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

func (b *Blockchain) processEpochTransition(newState *primitives.State) error {
	activeValidatorIndices := primitives.GetActiveValidatorIndices(newState.ValidatorRegistry)
	totalBalance := newState.GetTotalBalance(activeValidatorIndices, b.config)
	currentEpochAttestations := []primitives.PendingAttestation{}
	for _, a := range newState.LatestAttestations {
		if newState.Slot-b.config.EpochLength <= a.Data.Slot && a.Data.Slot < newState.Slot {
			currentEpochAttestations = append(currentEpochAttestations, a)
		}
	}

	previousEpochBoundaryHash, err := b.GetNodeByHeight(newState.Slot - b.config.EpochLength)
	if err != nil {
		return err
	}

	currentEpochBoundaryAttestations := []primitives.PendingAttestation{}
	for _, a := range currentEpochAttestations {
		if a.Data.EpochBoundaryHash.IsEqual(&previousEpochBoundaryHash) && a.Data.JustifiedSlot == newState.JustifiedSlot {
			currentEpochBoundaryAttestations = append(currentEpochBoundaryAttestations, a)
		}
	}
	currentEpochBoundaryAttesterIndices := map[uint32]struct{}{}
	for _, a := range currentEpochBoundaryAttestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, b.config)
		if err != nil {
			return err
		}
		for _, p := range participants {
			currentEpochBoundaryAttesterIndices[p] = struct{}{}
		}
	}
	currentEpochBoundaryAttestingBalance := newState.GetTotalBalanceMap(currentEpochBoundaryAttesterIndices, b.config)

	previousEpochAttestations := []primitives.PendingAttestation{}
	for _, a := range newState.LatestAttestations {
		if newState.Slot-2*b.config.EpochLength <= a.Data.Slot && a.Data.Slot < newState.Slot-b.config.EpochLength {
			previousEpochAttestations = append(previousEpochAttestations, a)
		}
	}

	previousEpochAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochAttestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, b.config)
		if err != nil {
			return err
		}
		for _, p := range participants {
			previousEpochAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochJustifiedAttestations := []primitives.PendingAttestation{}
	for _, a := range currentEpochAttestations {
		if a.Data.JustifiedSlot == newState.PreviousJustifiedSlot {
			previousEpochJustifiedAttestations = append(previousEpochJustifiedAttestations, a)
		}
	}

	for _, a := range previousEpochAttestations {
		if a.Data.JustifiedSlot == newState.PreviousJustifiedSlot {
			previousEpochJustifiedAttestations = append(previousEpochJustifiedAttestations, a)
		}
	}

	previousEpochJustifiedAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochJustifiedAttestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, b.config)
		if err != nil {
			return err
		}
		for _, p := range participants {
			previousEpochJustifiedAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochJustifiedAttestingBalance := newState.GetTotalBalanceMap(previousEpochJustifiedAttesterIndices, b.config)

	epochBoundaryHashMinus2, err := b.GetNodeByHeight(newState.Slot - 2*b.config.EpochLength)
	if err != nil {
		return err
	}

	previousEpochBoundaryAttestations := []primitives.PendingAttestation{}

	for _, a := range previousEpochJustifiedAttestations {
		if epochBoundaryHashMinus2.IsEqual(&a.Data.EpochBoundaryHash) {
			previousEpochBoundaryAttestations = append(previousEpochBoundaryAttestations, a)
		}
	}

	previousEpochBoundaryAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochBoundaryAttestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, b.config)
		if err != nil {
			return err
		}
		for _, p := range participants {
			previousEpochBoundaryAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochBoundaryAttestingBalance := newState.GetTotalBalanceMap(previousEpochBoundaryAttesterIndices, b.config)

	previousEpochHeadAttestations := []primitives.PendingAttestation{}

	for _, a := range previousEpochAttestations {
		blockRoot, err := b.GetNodeByHeight(a.Data.Slot)
		if err != nil {
			return err
		}
		if a.Data.BeaconBlockHash.IsEqual(&blockRoot) {
			previousEpochHeadAttestations = append(previousEpochHeadAttestations, a)
		}
	}

	previousEpochHeadAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochHeadAttestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, b.config)
		if err != nil {
			return err
		}
		for _, p := range participants {
			previousEpochHeadAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochHeadAttestingBalance := newState.GetTotalBalanceMap(previousEpochHeadAttesterIndices, b.config)

	newState.PreviousJustifiedSlot = newState.JustifiedSlot
	newState.JustificationBitfield = newState.JustificationBitfield * 2
	if 3*previousEpochBoundaryAttestingBalance >= 2*totalBalance {
		newState.JustificationBitfield |= 2
		newState.JustifiedSlot = newState.Slot - 2*b.config.EpochLength
	}

	if 3*currentEpochBoundaryAttestingBalance >= 2*totalBalance {
		newState.JustificationBitfield |= 2
		newState.JustifiedSlot = newState.Slot - b.config.EpochLength
	}

	if (newState.PreviousJustifiedSlot == newState.Slot-2*b.config.EpochLength && newState.JustificationBitfield%4 == 3) ||
		(newState.PreviousJustifiedSlot == newState.Slot-3*b.config.EpochLength && newState.JustificationBitfield%8 == 7) ||
		(newState.PreviousJustifiedSlot == newState.Slot-4*b.config.EpochLength && newState.JustificationBitfield%16 > 14) {
		newState.FinalizedSlot = newState.PreviousJustifiedSlot
	}

	attestingValidatorIndices := func(shardComittee primitives.ShardAndCommittee, shardBlockRoot chainhash.Hash) ([]uint32, error) {
		out := []uint32{}
		for _, a := range currentEpochAttestations {
			if a.Data.Shard == shardComittee.Shard && a.Data.ShardBlockHash.IsEqual(&shardBlockRoot) {
				participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, b.config)
				if err != nil {
					return nil, err
				}
				for _, p := range participants {
					out = append(out, p)
				}
			}
		}
		return out, nil
	}

	winningRoot := func(shardCommittee primitives.ShardAndCommittee) (*chainhash.Hash, error) {
		balances := map[chainhash.Hash]struct{}{}
		largestBalance := uint64(0)
		largestBalanceHash := new(chainhash.Hash)
		for _, a := range currentEpochAttestations {
			if a.Data.Shard != shardCommittee.Shard {
				continue
			}
			if _, exists := balances[a.Data.ShardBlockHash]; !exists {
				participationIndices, err := attestingValidatorIndices(shardCommittee, a.Data.ShardBlockHash)
				if err != nil {
					return nil, err
				}
				balance := newState.GetTotalBalance(participationIndices, b.config)
				if balance > largestBalance {
					largestBalance = balance
					largestBalanceHash = &a.Data.ShardBlockHash
				}
			}
		}
		if largestBalance == 0 {
			return nil, errors.New("no attestations from correct shard ID")
		}
		return largestBalanceHash, nil
	}

	shardWinnerCache := make([]map[uint64]chainhash.Hash, len(newState.ShardAndCommitteeForSlots))

	for i, shardCommitteeAtSlot := range newState.ShardAndCommitteeForSlots {
		for _, shardCommittee := range shardCommitteeAtSlot {
			bestRoot, err := winningRoot(shardCommittee)
			if err != nil {
				return err
			}
			shardWinnerCache[i][shardCommittee.Shard] = *bestRoot
			attestingCommittee, err := attestingValidatorIndices(shardCommittee, *bestRoot)
			if err != nil {
				return err
			}
			totalAttestingBalance := newState.GetTotalBalance(attestingCommittee, b.config)
			totalBalance := newState.GetTotalBalance(shardCommittee.Committee, b.config)

			if 3*totalAttestingBalance >= 2*totalBalance {
				newState.LatestCrosslinks[shardCommittee.Shard] = primitives.Crosslink{
					Slot:           newState.Slot,
					ShardBlockHash: *bestRoot,
				}
			}
		}
	}

	baseRewardQuotient := b.config.BaseRewardQuotient * intSqrt(totalBalance)
	baseReward := func(index uint32) uint64 {
		return newState.GetEffectiveBalance(index, b.config) / baseRewardQuotient / 5
	}

	inactivityPenalty := func(index uint32, epochsSinceFinality uint64) uint64 {
		return baseReward(index) + newState.GetEffectiveBalance(index, b.config)*epochsSinceFinality/b.config.InactivityPenaltyQuotient/2
	}

	epochsSinceFinality := (newState.Slot - newState.FinalizedSlot) / b.config.EpochLength

	validatorAttestationCache := map[uint32]*primitives.PendingAttestation{}
	for _, a := range currentEpochAttestations {
		participation, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, b.config)
		if err != nil {
			return err
		}

		for _, p := range participation {
			validatorAttestationCache[p] = &a
		}
	}

	if epochsSinceFinality <= 4 {
		// any validator in previous_epoch_justified_attester_indices is rewarded
		for index := range previousEpochJustifiedAttesterIndices {
			newState.ValidatorBalances[index] += baseReward(index) * previousEpochJustifiedAttestingBalance / totalBalance
		}

		// any validator in previous_epoch_boundary_attester_indices is rewarded
		for index := range previousEpochBoundaryAttesterIndices {
			newState.ValidatorBalances[index] += baseReward(index) * previousEpochBoundaryAttestingBalance / totalBalance
		}

		// any validator in previous_epoch_head_attester_indices is rewarded
		for index := range previousEpochHeadAttesterIndices {
			newState.ValidatorBalances[index] += baseReward(index) * previousEpochHeadAttestingBalance / totalBalance
		}

		// any validator in previous_epoch_head_attester_indices is rewarded
		for index := range previousEpochAttesterIndices {
			inclusionDistance := validatorAttestationCache[index].SlotIncluded - validatorAttestationCache[index].Data.Slot
			newState.ValidatorBalances[index] += baseReward(index) * b.config.MinAttestationInclusionDelay / inclusionDistance
		}

		// any validator not in previous_epoch_head_attester_indices is slashed
		// any validator not in previous_epoch_boundary_attester_indices is slashed
		// any validator not in previous_epoch_justified_attester_indices is slashed
		for idx, validator := range newState.ValidatorRegistry {
			index := uint32(idx)
			if validator.Status != primitives.Active {
				continue
			}
			if _, found := previousEpochHeadAttesterIndices[index]; !found {
				newState.ValidatorBalances[index] -= baseReward(index)
			}
			if _, found := previousEpochBoundaryAttesterIndices[index]; !found {
				newState.ValidatorBalances[index] -= baseReward(index)
			}
			if _, found := previousEpochJustifiedAttesterIndices[index]; !found {
				newState.ValidatorBalances[index] -= baseReward(index)
			}
		}
	} else {
		// any validator not in previous_epoch_head_attester_indices is slashed
		for idx, validator := range newState.ValidatorRegistry {
			index := uint32(idx)
			if validator.Status == primitives.Active {
				if _, found := previousEpochJustifiedAttesterIndices[index]; !found {
					newState.ValidatorBalances[index] -= inactivityPenalty(index, epochsSinceFinality)
				}
				if _, found := previousEpochBoundaryAttesterIndices[index]; !found {
					newState.ValidatorBalances[index] -= inactivityPenalty(index, epochsSinceFinality)
				}
				if _, found := previousEpochHeadAttesterIndices[index]; !found {
					newState.ValidatorBalances[index] -= baseReward(index)
				}
			} else if validator.Status == primitives.ExitedWithPenalty {
				newState.ValidatorBalances[index] -= inactivityPenalty(index, epochsSinceFinality) + baseReward(index)
			}
			// inclusion delay penalty
			if _, found := previousEpochAttesterIndices[index]; !found {
				inclusionDistance := validatorAttestationCache[index].SlotIncluded - validatorAttestationCache[index].Data.Slot
				newState.ValidatorBalances[index] -= baseReward(index) - baseReward(index)*b.config.MinAttestationInclusionDelay/inclusionDistance
			}
		}
	}

	for index := range previousEpochAttesterIndices {
		proposerIndex := newState.GetBeaconProposerIndex(validatorAttestationCache[index].SlotIncluded, b.config)
		newState.ValidatorBalances[proposerIndex] += baseReward(index) / b.config.IncluderRewardQuotient
	}

	for slot, shardCommitteeAtSlot := range newState.ShardAndCommitteeForSlots[:b.config.EpochLength] {
		for _, shardCommittee := range shardCommitteeAtSlot {
			winningRoot := shardWinnerCache[slot][shardCommittee.Shard]
			participationIndices, err := attestingValidatorIndices(shardCommittee, winningRoot)
			if err != nil {
				return err
			}

			participationIndicesMap := map[uint32]struct{}{}
			for _, p := range participationIndices {
				participationIndicesMap[p] = struct{}{}
			}

			totalAttestingBalance := newState.GetTotalBalance(participationIndices, b.config)
			totalBalance := newState.GetTotalBalance(shardCommittee.Committee, b.config)

			for _, index := range shardCommittee.Committee {
				if _, found := participationIndicesMap[index]; found {
					newState.ValidatorBalances[index] += baseReward(index) * totalAttestingBalance / totalBalance
				} else {
					newState.ValidatorBalances[index] -= baseReward(index)
				}
			}
		}
	}

	for index, validator := range newState.ValidatorRegistry {
		if validator.Status == primitives.Active && newState.ValidatorBalances[index] < b.config.EjectionBalance {
			newState.UpdateValidatorStatus(uint32(index), primitives.ExitedWithoutPenalty, b.config)
		}
	}

	newState.UpdateValidatorRegistry(b.config)

	// FINAL UPDATES

	return nil
}

// ApplyAttestation verifies and applies an attestation to the given state.
func (b *Blockchain) ApplyAttestation(s *primitives.State, att primitives.Attestation, c *config.Config) error {
	if att.Data.Slot+c.MinAttestationInclusionDelay > s.Slot {
		return errors.New("attestation included too soon")
	}

	if att.Data.Slot+c.EpochLength < s.Slot {
		return errors.New("attestation was not included within 1 epoch")
	}

	expectedJustifiedSlot := s.JustifiedSlot
	if att.Data.Slot < s.Slot-(s.Slot%c.EpochLength) {
		expectedJustifiedSlot = s.PreviousJustifiedSlot
	}

	if att.Data.JustifiedSlot != expectedJustifiedSlot {
		return errors.New("justified slot did not match expected justified slot")
	}

	node, err := b.GetNodeByHeight(att.Data.JustifiedSlot)
	if err != nil {
		return err
	}

	if !att.Data.JustifiedBlockHash.IsEqual(&node) {
		return errors.New("justified block hash did not match")
	}

	if len(s.LatestCrosslinks) <= int(att.Data.Shard) {
		return errors.New("invalid shard number")
	}

	latestCrosslinkRoot := s.LatestCrosslinks[att.Data.Shard].ShardBlockHash

	if !att.Data.LatestCrosslinkHash.IsEqual(&latestCrosslinkRoot) && !att.Data.ShardBlockHash.IsEqual(&latestCrosslinkRoot) {
		return errors.New("latest crosslink is invalid")
	}

	participants, err := s.GetAttestationParticipants(att.Data, att.ParticipationBitfield, c)
	if err != nil {
		return err
	}

	groupPublicKey := bls.NewAggregatePublicKey()
	for _, p := range participants {
		groupPublicKey.AggregatePubKey(&s.ValidatorRegistry[p].Pubkey)
	}

	dataRoot, err := ssz.TreeHash(primitives.AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
	if err != nil {
		return err
	}

	valid, err := bls.VerifySig(groupPublicKey, dataRoot[:], &att.AggregateSig, primitives.GetDomain(s.ForkData, att.Data.Slot, bls.DomainAttestation))
	if err != nil {
		return err
	}

	if !valid {
		return errors.New("attestation signature is invalid")
	}

	// REMOVEME
	if !att.Data.ShardBlockHash.IsEqual(&zeroHash) {
		return errors.New("invalid block hash")
	}

	s.LatestAttestations = append(s.LatestAttestations, primitives.PendingAttestation{
		Data:                  att.Data,
		ParticipationBitfield: att.ParticipationBitfield,
		CustodyBitfield:       att.CustodyBitfield,
		SlotIncluded:          s.Slot,
	})
	return nil
}

// ProcessBlock is called when a block is received from a peer.
func (b *Blockchain) ProcessBlock(block *primitives.Block) error {
	err := b.ApplyBlock(block)
	if err != nil {
		return err
	}

	err = b.AddBlock(block)
	if err != nil {
		return err
	}

	return nil
}

// GetNewRecentBlockHashes will take a list of recent block hashes and
// shift them to the right, filling them in with the parentHash provided.
func GetNewRecentBlockHashes(oldHashes []chainhash.Hash, parentSlot uint64, currentSlot uint64, parentHash chainhash.Hash) []chainhash.Hash {
	d := currentSlot - parentSlot
	newHashes := oldHashes[:]
	numberToAdd := int(d)
	if numberToAdd > len(oldHashes) {
		numberToAdd = len(oldHashes)
	}
	for i := 0; i < numberToAdd; i++ {
		newHashes = append(newHashes, parentHash)
	}
	return newHashes
}

// GetState gets a copy of the current state of the blockchain.
func (b *Blockchain) GetState() primitives.State {
	b.stateLock.Lock()
	state := b.state
	b.stateLock.Unlock()
	return state
}
