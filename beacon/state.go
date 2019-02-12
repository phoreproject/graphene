package beacon

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/phoreproject/prysm/shared/ssz"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	logger "github.com/sirupsen/logrus"
)

// InitializeState initializes state to the genesis state according to the config.
func (b *Blockchain) InitializeState(initialValidators []InitialValidatorEntry, genesisTime uint64, skipValidation bool) error {
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
		validatorIndex, err := b.state.ProcessDeposit(deposit.PubKey, deposit.DepositSize, deposit.ProofOfPossession, deposit.WithdrawalCredentials, skipValidation, b.config)
		if err != nil {
			return err
		}
		if b.state.GetEffectiveBalance(validatorIndex, b.config) == b.config.MaxDeposit*config.UnitInCoin {
			err := b.state.UpdateValidatorStatus(validatorIndex, primitives.Active, b.config)
			if err != nil {
				return err
			}
		}
	}

	initialShuffling := GetNewShuffling(zeroHash, b.state.ValidatorRegistry, 0, b.config)
	b.state.ShardAndCommitteeForSlots = append(initialShuffling, initialShuffling...)

	stateRoot, err := ssz.TreeHash(b.state)
	if err != nil {
		return err
	}

	block0 := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   0,
			StateRoot:    stateRoot,
			ParentRoot:   zeroHash,
			RandaoReveal: bls.EmptySignature.Serialize(),
			Signature:    bls.EmptySignature.Serialize(),
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

	blockHash, err := ssz.TreeHash(block0)
	if err != nil {
		return err
	}

	err = b.db.SetBlock(block0)
	if err != nil {
		return err
	}
	node, err := b.addBlockNodeToIndex(&block0, blockHash)
	if err != nil {
		return err
	}

	b.chain.finalizedHead = blockNodeAndState{node, b.state}
	b.chain.justifiedHead = blockNodeAndState{node, b.state}
	b.chain.tip = node
	b.stateMap = make(map[chainhash.Hash]primitives.State)
	b.stateMap[blockHash] = b.state

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

// AddBlock adds a block header to the current chain. The block should already
// have been validated by this point.
func (b *Blockchain) AddBlock(block *primitives.Block) error {
	blockHash, err := ssz.TreeHash(block)
	if err != nil {
		return err
	}

	logger.WithField("hash", blockHash).Debug("adding block to cache and updating head if needed")

	err = b.db.SetBlock(*block)
	if err != nil {
		return err
	}

	return nil
}

// ApplyBlock tries to apply a block to the state.
func (b *Blockchain) ApplyBlock(block *primitives.Block) (*primitives.State, error) {
	// copy the state so we can easily revert
	newState := b.state.Copy()

	// increase the slot number
	newState.Slot++

	previousBlockRoot, err := b.GetHashByHeight(block.BlockHeader.SlotNumber - 1)
	if err != nil {
		return nil, err
	}

	newState.LatestBlockHashes[(newState.Slot-1)%b.config.LatestBlockRootsLength] = previousBlockRoot

	if newState.Slot%b.config.LatestBlockRootsLength == 0 {
		latestBlockHashesRoot, err := ssz.TreeHash(newState.LatestBlockHashes)
		if err != nil {
			return nil, err
		}
		newState.BatchedBlockRoots = append(newState.BatchedBlockRoots, latestBlockHashesRoot)
	}

	if block.BlockHeader.SlotNumber != newState.Slot {
		return nil, errors.New("block has incorrect slot number")
	}

	blockWithoutSignature := block.Copy()
	blockWithoutSignature.BlockHeader.Signature = bls.EmptySignature.Serialize()
	blockWithoutSignatureRoot, err := ssz.TreeHash(blockWithoutSignature)
	if err != nil {
		return nil, err
	}

	proposal := primitives.ProposalSignedData{
		Slot:      newState.Slot,
		Shard:     b.config.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot, err := ssz.TreeHash(proposal)
	if err != nil {
		return nil, err
	}

	beaconProposerIndex := newState.GetBeaconProposerIndex(newState.Slot, b.config)

	proposerPub, err := bls.DeserializePublicKey(newState.ValidatorRegistry[beaconProposerIndex].Pubkey)
	if err != nil {
		return nil, err
	}

	proposerSig, err := bls.DeserializeSignature(block.BlockHeader.Signature)
	if err != nil {
		return nil, err
	}

	valid, err := bls.VerifySig(proposerPub, proposalRoot[:], proposerSig, bls.DomainProposal)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, errors.New("block had invalid signature")
	}

	proposer := &newState.ValidatorRegistry[beaconProposerIndex]

	var proposerSlotsBytes [8]byte
	binary.BigEndian.PutUint64(proposerSlotsBytes[:], proposer.ProposerSlots)

	randaoSig, err := bls.DeserializeSignature(block.BlockHeader.RandaoReveal)
	if err != nil {
		return nil, err
	}

	valid, err = bls.VerifySig(proposerPub, proposerSlotsBytes[:], randaoSig, bls.DomainRandao)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errors.New("block has invalid randao signature")
	}

	// increase the randao skips of the proposer
	newState.ValidatorRegistry[newState.GetBeaconProposerIndex(newState.Slot, b.config)].ProposerSlots++

	randaoRevealSerialized, err := ssz.TreeHash(block.BlockHeader.RandaoReveal)
	if err != nil {
		return nil, err
	}

	for i := range newState.RandaoMix {
		newState.RandaoMix[i] ^= randaoRevealSerialized[i]
	}

	if len(block.BlockBody.ProposerSlashings) > b.config.MaxProposerSlashings {
		return nil, errors.New("more than maximum proposer slashings")
	}

	if len(block.BlockBody.CasperSlashings) > b.config.MaxCasperSlashings {
		return nil, errors.New("more than maximum casper slashings")
	}

	if len(block.BlockBody.Attestations) > b.config.MaxAttestations {
		return nil, errors.New("more than maximum casper slashings")
	}

	for _, s := range block.BlockBody.ProposerSlashings {
		err := newState.ApplyProposerSlashing(s, b.config)
		if err != nil {
			return nil, err
		}
	}

	for _, c := range block.BlockBody.CasperSlashings {
		err := newState.ApplyCasperSlashing(c, b.config)
		if err != nil {
			return nil, err
		}
	}

	for _, a := range block.BlockBody.Attestations {
		err := b.ApplyAttestation(&newState, a, b.config)
		if err != nil {
			return nil, err
		}
	}

	// process deposits here

	for _, e := range block.BlockBody.Exits {
		newState.ApplyExit(e, b.config)
	}

	if newState.Slot%b.config.EpochLength == 0 {
		err := b.processEpochTransition(&newState)
		if err != nil {
			return nil, err
		}
	}

	// VERIFY BLOCK STATE ROOT MATCHES STATE ROOT FROM PREVIOUS BLOCK IF NEEDED
	return &newState, nil
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

	previousEpochBoundaryHash, err := b.GetHashByHeight(newState.Slot - b.config.EpochLength)
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

	epochBoundaryHashMinus2 := chainhash.Hash{}
	if newState.Slot >= 2*b.config.EpochLength {
		ebhm2, err := b.GetHashByHeight(newState.Slot - 2*b.config.EpochLength)
		epochBoundaryHashMinus2 = ebhm2
		if err != nil {
			return err
		}
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
		blockRoot, err := b.GetHashByHeight(a.Data.Slot)
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
		newState.JustificationBitfield |= 1
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
			balances[a.Data.ShardBlockHash] = struct{}{}
		}
		for _, a := range currentEpochAttestations {
			if a.Data.Shard != shardCommittee.Shard {
				continue
			}
			if _, exists := balances[a.Data.ShardBlockHash]; exists {
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
			if shardWinnerCache[i] == nil {
				shardWinnerCache[i] = make(map[uint64]chainhash.Hash)
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

	baseRewardQuotient := b.config.BaseRewardQuotient * intSqrt(totalBalance*config.UnitInCoin)
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

	shouldUpdateRegistry := true

	if newState.FinalizedSlot <= newState.ValidatorRegistryLatestChangeSlot {
		shouldUpdateRegistry = false
	}

	for _, shardAndCommittees := range newState.ShardAndCommitteeForSlots {
		for _, committee := range shardAndCommittees {
			if newState.LatestCrosslinks[committee.Shard].Slot <= newState.ValidatorRegistryLatestChangeSlot {
				shouldUpdateRegistry = false
				goto done
			}
		}
	}
done:

	if shouldUpdateRegistry {
		newState.UpdateValidatorRegistry(b.config)

		newState.ValidatorRegistryLatestChangeSlot = newState.Slot
		copy(newState.ShardAndCommitteeForSlots[:b.config.EpochLength], newState.ShardAndCommitteeForSlots[b.config.EpochLength:])
		lastSlot := newState.ShardAndCommitteeForSlots[len(newState.ShardAndCommitteeForSlots)-1]
		lastCommittee := lastSlot[len(lastSlot)-1]
		nextStartShard := (lastCommittee.Shard + 1) % uint64(b.config.ShardCount)
		newShuffling := GetNewShuffling(newState.RandaoMix, newState.ValidatorRegistry, int(nextStartShard), b.config)
		copy(newState.ShardAndCommitteeForSlots[b.config.EpochLength:], newShuffling)
	} else {
		copy(newState.ShardAndCommitteeForSlots[:b.config.EpochLength], newState.ShardAndCommitteeForSlots[b.config.EpochLength:])
		epochsSinceLastRegistryChange := (newState.Slot - newState.ValidatorRegistryLatestChangeSlot) / b.config.EpochLength
		startShard := newState.ShardAndCommitteeForSlots[0][0].Shard

		// epochsSinceLastRegistryChange is a power of 2
		if epochsSinceLastRegistryChange&(epochsSinceLastRegistryChange-1) == 0 {
			newShuffling := GetNewShuffling(newState.RandaoMix, newState.ValidatorRegistry, int(startShard), b.config)
			copy(newState.ShardAndCommitteeForSlots[b.config.EpochLength:], newShuffling)
		}
	}

	newLatestAttestations := []primitives.PendingAttestation{}
	for _, a := range newState.LatestAttestations {
		if a.Data.Slot >= newState.Slot-b.config.EpochLength {
			newLatestAttestations = append(newLatestAttestations, a)
		}
	}

	newState.LatestAttestations = newLatestAttestations

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

	node, err := b.GetHashByHeight(att.Data.JustifiedSlot)
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
		p, err := bls.DeserializePublicKey(s.ValidatorRegistry[p].Pubkey)
		if err != nil {
			return err
		}
		groupPublicKey.AggregatePubKey(p)
	}

	dataRoot, err := ssz.TreeHash(primitives.AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
	if err != nil {
		return err
	}

	aggSig, err := bls.DeserializeSignature(att.AggregateSig)
	if err != nil {
		return err
	}

	valid, err := bls.VerifySig(groupPublicKey, dataRoot[:], aggSig, primitives.GetDomain(s.ForkData, att.Data.Slot, bls.DomainAttestation))
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
	// VALIDATE BLOCK HERE

	blockHash, err := ssz.TreeHash(block)
	if err != nil {
		return err
	}

	logger.WithField("hash", blockHash).Debug("processing new block")

	err = b.AddBlock(block)
	if err != nil {
		return err
	}

	node, err := b.addBlockNodeToIndex(block, blockHash)
	if err != nil {
		return err
	}

	logger.WithField("hash", blockHash).Debug("applying block")

	newState, err := b.ApplyBlock(block)
	if err != nil {
		return err
	}

	b.stateMap[blockHash] = *newState

	logger.WithField("hash", blockHash).Debug("updating chain head")

	err = b.UpdateChainHead(block)
	if err != nil {
		return err
	}

	finalizedNode, err := getAncestor(node, newState.FinalizedSlot)
	if err != nil {
		return err
	}
	finalizedState := b.stateMap[finalizedNode.hash]
	finalizedNodeAndState := blockNodeAndState{finalizedNode, finalizedState}
	b.chain.finalizedHead = finalizedNodeAndState

	justifiedNode, err := getAncestor(node, newState.JustifiedSlot)
	if err != nil {
		return err
	}
	justifiedState := b.stateMap[justifiedNode.hash]
	justifiedNodeAndState := blockNodeAndState{justifiedNode, justifiedState}
	b.chain.justifiedHead = justifiedNodeAndState

	// once we finalize the block, we can get rid of any states before finalizedSlot
	for i := range b.stateMap {
		// if it happened before finalized slot, we don't need it
		if b.stateMap[i].Slot <= b.state.FinalizedSlot {
			delete(b.stateMap, i)
		}
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
