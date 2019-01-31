package beacon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/primitives"
	"github.com/phoreproject/synapse/bls"
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

	block0 := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   0,
			StateRoot:    b.state.Hash(),
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

	err := b.AddBlock(&block0)
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
	logger.WithField("hash", block.Hash()).Debug("adding block to cache and updating head if needed")
	err := b.UpdateChainHead(block)
	if err != nil {
		return err
	}

	err = b.db.SetBlock(*block)
	if err != nil {
		return err
	}

	return nil
}

// ApplyBlock tries to apply a block to the state.
func (b *Blockchain) ApplyBlock(block *primitives.Block) error {
	// copy the state so we can easily revert
	newState := b.state.Copy()

	// increase the slot number
	newState.Slot++

	// increase the randao skips of the proposer
	newState.ValidatorRegistry[b.state.GetBeaconProposerIndex(b.state.Slot, b.config)].RandaoSkips++

	previousBlockRoot, err := b.GetNodeByHeight(block.SlotNumber - 1)
	if err != nil {
		return err
	}

	newState.LatestBlockHashes[(newState.Slot-1)%b.config.LatestBlockRootsLength] = previousBlockRoot

	if newState.Slot%b.config.LatestBlockRootsLength == 0 {
		b.state.BatchedBlockRoots = append(b.state.BatchedBlockRoots, primitives.MerkleRootHash(newState.LatestBlockHashes))
	}

	if block.SlotNumber != newState.Slot {
		return errors.New("block has incorrect slot number")
	}

	blockWithoutSignature := block.Copy()
	blockWithoutSignature.Signature = *bls.EmptySignature
	blockWithoutSignatureRoot := blockWithoutSignature.Hash()

	proposalRoot := primitives.ProposalSignedData{
		Slot: newState.Slot,
		Shard: b.config.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot
	}

	beaconProposerIndex := newState.GetBeaconProposerIndex()
	bls.VerifySig(newState.ValidatorRegistry[beaconProposerIndex].Pubkey, proposalRoot.Hash())
	b.state = newState
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
