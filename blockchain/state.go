package blockchain

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/serialization"
	"github.com/phoreproject/synapse/transaction"
)

// InitializeState initializes state to the genesis state according to the config.
func (b Blockchain) InitializeState(initialValidators []InitialValidatorEntry) {
	b.state.Crystallized.Validators = make([]primitives.Validator, len(initialValidators))
	for _, v := range initialValidators {
		b.AddValidator(b.state.Crystallized.Validators, v.PubKey, v.ProofOfPossession, v.WithdrawalShard, v.WithdrawalAddress, v.RandaoCommitment, 0)
	}

	x := b.GetNewShuffling(zeroHash, 0)

	crosslinks := make([]primitives.Crosslink, b.config.ShardCount)

	for i := 0; i < b.config.ShardCount; i++ {
		crosslinks[i] = primitives.Crosslink{
			RecentlyChanged: false,
			Slot:            0,
			Hash:            &zeroHash,
		}
	}

	b.state.Crystallized = primitives.CrystallizedState{
		Validators:                  b.state.Crystallized.Validators,
		ValidatorSetChangeSlot:      0,
		Crosslinks:                  crosslinks,
		LastStateRecalculation:      0,
		LastFinalizedSlot:           0,
		LastJustifiedSlot:           0,
		JustifiedStreak:             0,
		ShardAndCommitteeForSlots:   append(x, x...),
		DepositsPenalizedInPeriod:   []uint32{},
		ValidatorSetDeltaHashChange: zeroHash,
		PreForkVersion:              InitialForkVersion,
		PostForkVersion:             InitialForkVersion,
		ForkSlotNumber:              0,
	}

	recentBlockHashes := make([]chainhash.Hash, b.config.CycleLength*2)
	for i := 0; i < b.config.CycleLength*2; i++ {
		recentBlockHashes[i] = zeroHash
	}

	b.state.Active = primitives.ActiveState{
		PendingActions:      []transaction.Transaction{},
		PendingAttestations: []transaction.Attestation{},
		RecentBlockHashes:   recentBlockHashes,
		RandaoMix:           zeroHash,
		Balances:            make(map[serialization.Address]uint64),
	}
}

// MinEmptyValidator finds the first validator slot that is empty.
func MinEmptyValidator(validators []primitives.Validator) int {
	for i, v := range validators {
		if v.Status == Withdrawn {
			return i
		}
	}
	return -1
}

// AddValidator adds a validator to the current validator set.
func (b *Blockchain) AddValidator(currentValidators []primitives.Validator, pubkey []byte, proofOfPossession []byte, withdrawalShard uint32, withdrawalAddress serialization.Address, randaoCommitment chainhash.Hash, currentSlot uint64) (uint32, error) {
	verifies := true // blsig.Verify(chainhash.HashB(pubkey), pubkey, proofOfPossession)
	if !verifies {
		return 0, errors.New("validator proof of possesion does not verify")
	}

	var pk [32]byte
	copy(pk[:], pubkey)

	rec := primitives.Validator{
		Pubkey:            pk,
		WithdrawalAddress: withdrawalAddress,
		WithdrawalShardID: withdrawalShard,
		RandaoCommitment:  randaoCommitment,
		Balance:           b.config.DepositSize,
		Status:            PendingActivation,
		ExitSlot:          0,
	}

	index := MinEmptyValidator(currentValidators)
	if index == -1 {
		currentValidators = append(currentValidators, rec)
		return uint32(len(currentValidators) - 1), nil
	}
	currentValidators[index] = rec
	return uint32(index), nil
}

// GetActiveValidatorIndices gets the indices of active validators.
func (b *Blockchain) GetActiveValidatorIndices() []uint32 {
	var l []uint32
	for i, v := range b.state.Crystallized.Validators {
		if v.Status == Active {
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

// GetNewShuffling calculates the new shuffling of validators
// to slots and shards.
func (b *Blockchain) GetNewShuffling(seed chainhash.Hash, crosslinkingStart int) [][]primitives.ShardAndCommittee {
	activeValidators := b.GetActiveValidatorIndices()
	numActiveValidators := len(activeValidators)

	committeesPerSlot := numActiveValidators/b.config.CycleLength/(b.config.MinCommitteeSize*2) + 1
	// clamp between 1 and b.config.ShardCount / b.config.CycleLength
	if committeesPerSlot < 1 {
		committeesPerSlot = 1
	} else if committeesPerSlot > b.config.ShardCount/b.config.CycleLength {
		committeesPerSlot = b.config.ShardCount / b.config.CycleLength
	}

	output := make([][]primitives.ShardAndCommittee, b.config.CycleLength)

	shuffledValidatorIndices := ShuffleValidators(activeValidators, seed)

	validatorsPerSlot := Split(shuffledValidatorIndices, uint32(b.config.CycleLength))

	for slot, slotIndices := range validatorsPerSlot {
		shardIndices := Split(slotIndices, uint32(committeesPerSlot))

		shardIDStart := crosslinkingStart + slot*committeesPerSlot

		shardCommittees := make([]primitives.ShardAndCommittee, len(shardIndices))
		for shardPosition, indices := range shardIndices {
			shardCommittees[shardPosition] = primitives.ShardAndCommittee{
				ShardID:   uint32((shardIDStart + shardPosition) % b.config.ShardCount),
				Committee: indices,
			}
		}

		output[slot] = shardCommittees
	}
	return output
}

// ValidateAttestation checks attestation invariants and the BLS signature.
func (b Blockchain) ValidateAttestation(attestation transaction.Attestation, block primitives.BlockHeader, parentBlock primitives.BlockHeader, c Config) error {
	if attestation.Slot > parentBlock.SlotNumber {
		return errors.New("attestation slot number too high")
	}

	if !(attestation.Slot >= uint64(math.Max(float64(parentBlock.SlotNumber-uint64(c.CycleLength)+1), 0))) {
		return errors.New("attestation slot number too low")
	}

	if attestation.JustifiedSlot > b.state.Crystallized.LastJustifiedSlot {
		return errors.New("last justified slot should be less than or equal to the crystallized slot")
	}

	justifiedBlock, err := b.index.GetBlockNodeByHash(attestation.JustifiedBlockHash)
	if err != nil {
		return errors.New("justified block not in index")
	}

	if justifiedBlock.SlotNumber != attestation.Slot {
		return errors.New("justified slot does not match attestation")
	}

	// if (!len(attestation.att))

	// TODO: validate BLS sig

	return nil
}

// AddBlock adds a block header to the current chain. The block should already
// have been validated by this point.
func (b *Blockchain) AddBlock(h primitives.BlockHeader) error {
	var parent *BlockNode
	if !(h.AncestorHashes[0] == zeroHash && len(b.chain) == 0) {
		p, err := b.index.GetBlockNodeByHash(h.AncestorHashes[0])
		parent = p
		if err != nil {
			return err
		}
	}

	height := uint64(0)
	if parent != nil {
		height = parent.Height + 1
	}

	node := &BlockNode{BlockHeader: h, PrevNode: parent, Height: height}

	// Add block to the index
	b.index.AddNode(node)

	b.UpdateChainHead(node)

	return nil
}

// ProcessBlock is called when a block is received from a peer.
func (b Blockchain) ProcessBlock(block primitives.Block) error {
	err := b.ValidateIncomingBlock(block)
	if err != nil {
		return err
	}

	b.AddBlock(block.BlockHeader)

	return nil
}

// GetNewRecentBlockHashes will take a list of recent block hashes and
// shift them to the right, filling them in with the parentHash provided.
func GetNewRecentBlockHashes(oldHashes []*chainhash.Hash, parentSlot uint32, currentSlot uint32, parentHash *chainhash.Hash) []*chainhash.Hash {
	d := currentSlot - parentSlot
	newHashes := oldHashes[d:]
	numberToAdd := int(d)
	if numberToAdd > len(oldHashes) {
		numberToAdd = len(oldHashes)
	}
	for i := 0; i < numberToAdd; i++ {
		newHashes = append(newHashes, parentHash)
	}
	return newHashes
}

// UpdateAncestorHashes fills in the parent hash in ancestor hashes
// where the ith element represents the 2**i past block.
func UpdateAncestorHashes(parentAncestorHashes []chainhash.Hash, parentSlotNumber uint64, parentHash chainhash.Hash) []chainhash.Hash {
	newAncestorHashes := parentAncestorHashes[:]
	for i := uint(0); i < 32; i++ {
		if parentSlotNumber%(1<<i) == 0 {
			newAncestorHashes[i] = parentHash
		}
	}
	return newAncestorHashes
}

// GetShardsAndCommitteesForSlot gets the committee for each shard.
func (b Blockchain) GetShardsAndCommitteesForSlot(slot uint64) []primitives.ShardAndCommittee {
	earliestSlotInArray := b.state.Crystallized.LastStateRecalculation - uint64(b.config.CycleLength)
	return b.state.Crystallized.ShardAndCommitteeForSlots[slot-earliestSlotInArray]
}

func hasVoted(bitfield []byte, index int) bool {
	return bitfield[index/8]&(128>>uint(index%8)) != 0
}

func repeatHash(h chainhash.Hash, n int) chainhash.Hash {
	for n > 0 {
		h = chainhash.HashH(h[:])
		n--
	}
	return h
}

func (b Blockchain) getTotalActiveValidatorBalance() uint64 {
	total := uint64(0)
	for _, v := range b.state.Crystallized.Validators {
		if v.Status == Active {
			total += v.Balance
		}
	}
	return total
}

// ValidateIncomingBlock runs a couple of checks on an incoming block.
func (b Blockchain) ValidateIncomingBlock(newBlock primitives.Block) error {
	if len(newBlock.AncestorHashes) != 32 {
		return errors.New("ancestorHashes improperly formed")
	}

	parentBlock, err := b.index.GetBlockNodeByHash(newBlock.AncestorHashes[0])
	if err != nil {
		return err
	}

	newHashes := UpdateAncestorHashes(parentBlock.AncestorHashes, parentBlock.SlotNumber, parentBlock.Hash())
	for i := range newBlock.AncestorHashes {
		if newHashes[i] != newBlock.AncestorHashes[i] {
			return errors.New("ancestor hashes don't match expected value")
		}
	}

	for _, tx := range newBlock.Transactions {
		if sat, success := tx.Data.(transaction.SubmitAttestationTransaction); success {
			err := b.ValidateAttestation(sat.Attestation, newBlock.BlockHeader, parentBlock.BlockHeader, b.config)
			if err != nil {
				return err
			}
			b.state.Active.PendingAttestations = append(b.state.Active.PendingAttestations, sat.Attestation)
		}
		if _, success := tx.Data.(transaction.LoginTransaction); success {
			b.state.Active.PendingActions = append(b.state.Active.PendingActions, tx)
		}
		if _, success := tx.Data.(transaction.LogoutTransaction); success {
			b.state.Active.PendingActions = append(b.state.Active.PendingActions, tx)
		}
		if _, success := tx.Data.(transaction.RegisterTransaction); success {
			b.state.Active.PendingActions = append(b.state.Active.PendingActions, tx)
		}
	}

	shardAndCommittee := b.GetShardsAndCommitteesForSlot(parentBlock.SlotNumber)[0]
	proposerIndex := int(parentBlock.SlotNumber % uint64(len(shardAndCommittee.Committee)))

	// validate parent block proposer
	if newBlock.SlotNumber != 0 {
		attestations := []transaction.Attestation{}
		for _, a := range newBlock.Transactions {
			if sat, success := a.Data.(transaction.SubmitAttestationTransaction); success {
				attestations = append(attestations, sat.Attestation)
			}
		}
		if len(attestations) == 0 {
			return errors.New("invalid parent block proposer")
		}

		attestation := attestations[0]
		if attestation.ShardID != uint64(shardAndCommittee.ShardID) || attestation.Slot != parentBlock.SlotNumber || !hasVoted(attestation.AttesterBitField, proposerIndex) {
			return errors.New("invalid parent block proposer")
		}
	}

	validator := b.state.Crystallized.Validators[proposerIndex]

	expected := repeatHash(newBlock.RandaoReveal, int((newBlock.SlotNumber-validator.RandaoLastChange)/uint64(b.config.RandaoSlotsPerLayer)+1))

	if expected != validator.RandaoCommitment {
		return errors.New("randao does not match commitment")
	}

	for i := range b.state.Active.RandaoMix {
		b.state.Active.RandaoMix[i] ^= newBlock.RandaoReveal[i]
	}

	for newBlock.SlotNumber-b.state.Crystallized.LastStateRecalculation >= uint64(b.config.CycleLength) {
		for slot := b.state.Crystallized.LastStateRecalculation - uint64(b.config.CycleLength); slot < b.state.Crystallized.LastStateRecalculation-1; slot++ {
			shardsAndCommittees := b.GetShardsAndCommitteesForSlot(slot)
			committee := shardsAndCommittees[0].Committee
			totalBalance := uint64(0)
			for _, i := range committee {
				totalBalance += b.state.Crystallized.Validators[i].Balance
			}
			attesterBalance := uint64(0)

			node := b.chain[slot]
			block := b.db.GetBlockForHash(node.Hash())

			attestations := []transaction.Attestation{}
			for _, a := range block.Transactions {
				if sat, success := a.Data.(transaction.SubmitAttestationTransaction); success {
					attestations = append(attestations, sat.Attestation)
				}
			}
			if len(attestations) == 0 {
				return errors.New("invalid parent block proposer")
			}
			attestation := attestations[0]

			for _, i := range committee {
				if hasVoted(attestation.AttesterBitField, int(i)) {
					attesterBalance += b.state.Crystallized.Validators[i].Balance
				}
			}

			if 3*attesterBalance >= 2*totalBalance {
				if slot > b.state.Crystallized.LastJustifiedSlot {
					b.state.Crystallized.LastJustifiedSlot = slot
				}
				b.state.Crystallized.JustifiedStreak++
			} else {
				b.state.Crystallized.JustifiedStreak = 0
			}

			if b.state.Crystallized.JustifiedStreak >= uint64(b.config.CycleLength+1) {
				if b.state.Crystallized.LastFinalizedSlot < slot-uint64(b.config.CycleLength-1) {
					b.state.Crystallized.LastFinalizedSlot = slot - uint64(b.config.CycleLength-1)
				}
			}
		}
	}
	return nil
}
