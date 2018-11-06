package blockchain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	logger "github.com/inconshreveable/log15"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/serialization"
	"github.com/phoreproject/synapse/transaction"
)

// ActiveState is state that can change every block.
type ActiveState struct {
	PendingAttestations []transaction.Attestation
	PendingActions      []transaction.Transaction
	RecentBlockHashes   []chainhash.Hash
	RandaoMix           chainhash.Hash
	Balances            map[serialization.Address]uint64
}

// State is active and crystallized state.
type State struct {
	Active       ActiveState
	Crystallized CrystallizedState
}

// CrystallizedState is state that is updated every epoch
type CrystallizedState struct {
	ValidatorSetChangeSlot uint64
	Crosslinks             []primitives.Crosslink
	Validators             []primitives.Validator
	LastStateRecalculation uint64
	JustifiedStreak        uint64
	LastJustifiedSlot      uint64
	LastFinalizedSlot      uint64

	// ShardAndCommitteeForSlots is an array of slots where each element
	// is the committee assigned to that slot for each shard.
	ShardAndCommitteeForSlots   [][]primitives.ShardAndCommittee
	DepositsPenalizedInPeriod   []uint64
	ValidatorSetDeltaHashChange chainhash.Hash
	PreForkVersion              uint32
	PostForkVersion             uint32
	ForkSlotNumber              uint64
}

// Copy returns a copy of the crystallized state.
func (c CrystallizedState) Copy() CrystallizedState {
	return c
}

// ShardCommitteeByShardID gets the shards committee from a list of committees/shards
// in a list.
func ShardCommitteeByShardID(shardID uint32, shardCommittees []primitives.ShardAndCommittee) ([]uint32, error) {
	for _, s := range shardCommittees {
		if s.ShardID == shardID {
			return s.Committee, nil
		}
	}

	return nil, fmt.Errorf("unable to find committee based on shard: %v", shardID)
}

// CommitteeInShardAndSlot gets the committee of validator indices at a specific
// shard and slot given the relative slot number [0, CYCLE_LENGTH] and shard ID.
func CommitteeInShardAndSlot(slotIndex uint64, shardID uint32, shardCommittees [][]primitives.ShardAndCommittee) ([]uint32, error) {
	shardCommittee := shardCommittees[slotIndex]

	return ShardCommitteeByShardID(shardID, shardCommittee)
}

// GetAttesterIndices gets all of the validator indices involved with the committee
// assigned to the shard and slot of the committee.
func (c *CrystallizedState) GetAttesterIndices(attestation *transaction.Attestation, con *Config) ([]uint32, error) {
	slotsStart := c.LastStateRecalculation - uint64(con.CycleLength)
	slotIndex := (attestation.Slot - slotsStart) % uint64(con.CycleLength)
	return CommitteeInShardAndSlot(slotIndex, attestation.ShardID, c.ShardAndCommitteeForSlots)
}

// InitializeState initializes state to the genesis state according to the config.
func (b *Blockchain) InitializeState(initialValidators []InitialValidatorEntry) error {
	b.stateLock.Lock()
	validators := make([]primitives.Validator, len(initialValidators))
	for i, v := range initialValidators {
		validators[i] = primitives.Validator{
			Pubkey:            &v.PubKey,
			WithdrawalAddress: v.WithdrawalAddress,
			WithdrawalShardID: v.WithdrawalShard,
			RandaoCommitment:  v.RandaoCommitment,
			Balance:           b.config.DepositSize,
			Status:            Active,
			ExitSlot:          0,
		}
	}

	x := GetNewShuffling(zeroHash, validators, 0, b.config)

	crosslinks := make([]primitives.Crosslink, b.config.ShardCount)

	for i := 0; i < b.config.ShardCount; i++ {
		crosslinks[i] = primitives.Crosslink{
			RecentlyChanged: false,
			Slot:            0,
			Hash:            &zeroHash,
		}
	}

	b.state.Crystallized = CrystallizedState{
		Validators:                  validators,
		ValidatorSetChangeSlot:      0,
		Crosslinks:                  crosslinks,
		LastStateRecalculation:      0,
		LastFinalizedSlot:           0,
		LastJustifiedSlot:           0,
		JustifiedStreak:             0,
		ShardAndCommitteeForSlots:   append(x, x...),
		DepositsPenalizedInPeriod:   []uint64{},
		ValidatorSetDeltaHashChange: zeroHash,
		PreForkVersion:              InitialForkVersion,
		PostForkVersion:             InitialForkVersion,
		ForkSlotNumber:              0,
	}

	recentBlockHashes := make([]chainhash.Hash, b.config.CycleLength*2)
	for i := 0; i < b.config.CycleLength*2; i++ {
		recentBlockHashes[i] = zeroHash
	}

	b.state.Active = ActiveState{
		PendingActions:      []transaction.Transaction{},
		PendingAttestations: []transaction.Attestation{},
		RecentBlockHashes:   recentBlockHashes,
		RandaoMix:           zeroHash,
		Balances:            make(map[serialization.Address]uint64),
	}

	ancestorHashes := make([]chainhash.Hash, 32)

	block0 := primitives.Block{
		SlotNumber:            0,
		RandaoReveal:          zeroHash,
		AncestorHashes:        ancestorHashes,
		ActiveStateRoot:       zeroHash,
		CrystallizedStateRoot: zeroHash,
		Specials:              []transaction.Transaction{},
		Attestations:          []transaction.Attestation{},
	}

	b.stateLock.Unlock()

	b.AddBlock(&block0)

	return nil
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
func AddValidator(currentValidators []primitives.Validator, pubkey bls.PublicKey, proofOfPossession bls.Signature, withdrawalShard uint32, withdrawalAddress serialization.Address, randaoCommitment chainhash.Hash, currentSlot uint64, status uint8, con *Config) ([]primitives.Validator, uint32, error) {
	verifies, err := bls.VerifySig(&pubkey, pubkey.Hash(), &proofOfPossession)
	if err != nil || !verifies {
		return nil, 0, errors.New("validator proof of possesion does not verify")
	}

	rec := primitives.Validator{
		Pubkey:            &pubkey,
		WithdrawalAddress: withdrawalAddress,
		WithdrawalShardID: withdrawalShard,
		RandaoCommitment:  randaoCommitment,
		Balance:           con.DepositSize,
		Status:            status,
		ExitSlot:          0,
	}

	index := MinEmptyValidator(currentValidators)
	if index == -1 {
		currentValidators = append(currentValidators, rec)
		return currentValidators, uint32(len(currentValidators) - 1), nil
	}
	currentValidators[index] = rec
	return currentValidators, uint32(index), nil
}

// GetActiveValidatorIndices gets the indices of active validators.
func GetActiveValidatorIndices(vs []primitives.Validator) []uint32 {
	var l []uint32
	for i, v := range vs {
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
func GetNewShuffling(seed chainhash.Hash, validators []primitives.Validator, crosslinkingStart int, con *Config) [][]primitives.ShardAndCommittee {
	activeValidators := GetActiveValidatorIndices(validators)
	numActiveValidators := len(activeValidators)

	// clamp between 1 and b.config.ShardCount / b.config.CycleLength
	committeesPerSlot := clamp(1, con.ShardCount/con.CycleLength, numActiveValidators/con.CycleLength/(con.MinCommitteeSize*2)+1)

	output := make([][]primitives.ShardAndCommittee, con.CycleLength)

	shuffledValidatorIndices := ShuffleValidators(activeValidators, seed)

	validatorsPerSlot := Split(shuffledValidatorIndices, uint32(con.CycleLength))

	for slot, slotIndices := range validatorsPerSlot {
		shardIndices := Split(slotIndices, uint32(committeesPerSlot))

		shardIDStart := crosslinkingStart + slot*committeesPerSlot

		shardCommittees := make([]primitives.ShardAndCommittee, len(shardIndices))
		for shardPosition, indices := range shardIndices {
			shardCommittees[shardPosition] = primitives.ShardAndCommittee{
				ShardID:   uint32((shardIDStart + shardPosition) % con.ShardCount),
				Committee: indices,
			}
		}

		output[slot] = shardCommittees
	}
	return output
}

func checkTrailingZeros(a byte, numZeros uint8) bool {
	i := uint8(a)
	if i/128 == 1 && numZeros == 0 {
		return false
	}
	if (i%128)/64 == 1 && numZeros <= 1 {
		return false
	}
	if (i%64)/32 == 1 && numZeros <= 2 {
		return false
	}
	if (i%32)/16 == 1 && numZeros <= 3 {
		return false
	}
	if (i%16)/8 == 1 && numZeros <= 4 {
		return false
	}
	if (i%8)/4 == 1 && numZeros <= 5 {
		return false
	}
	if (i%4)/2 == 1 && numZeros <= 6 {
		return false
	}
	if (i%2) == 1 && numZeros <= 7 {
		return false
	}
	return true
}

// ValidateAttestation checks attestation invariants and the BLS signature.
func (b *Blockchain) ValidateAttestation(attestation *transaction.Attestation, parentBlock *primitives.Block, c *Config) error {
	if attestation.Slot > parentBlock.SlotNumber {
		return errors.New("attestation slot number too high")
	}

	// verify attestation slot >= max(parent.slot - CYCLE_LENGTH + 1, 0)
	if !(attestation.Slot >= uint64(math.Max(float64(parentBlock.SlotNumber-uint64(c.CycleLength)+1), 0))) {
		return errors.New("attestation slot number too low")
	}

	if attestation.JustifiedSlot > b.state.Crystallized.LastJustifiedSlot {
		return errors.New("last justified slot should be less than or equal to the crystallized slot")
	}

	justifiedBlockHash, err := b.chain.GetBlock(int(attestation.JustifiedSlot))
	if err != nil {
		return errors.New("justified block not in index")
	}

	if !justifiedBlockHash.IsEqual(&attestation.JustifiedBlockHash) {
		return errors.New("justified block hash does not match block at slot")
	}

	hashes := make([]chainhash.Hash, b.config.CycleLength-len(attestation.ObliqueParentHashes))

	for i := 1; i < b.config.CycleLength-len(attestation.ObliqueParentHashes)+1; i++ {
		h, err := b.chain.GetBlock(int(attestation.Slot) - b.config.CycleLength + i)
		if err != nil {
			return fmt.Errorf("could not find block at slot %d", int(attestation.Slot)-b.config.CycleLength+i)
		}

		hashes[i-1] = *h
	}

	attestationIndicesForShards := b.getShardsAndCommitteesForSlot(attestation.Slot)
	var attestationIndices primitives.ShardAndCommittee
	found := false
	for _, s := range attestationIndicesForShards {
		if s.ShardID == attestation.ShardID {
			attestationIndices = s
			found = true
		}
	}

	if !found {
		return fmt.Errorf("could not find shard id %d", attestation.ShardID)
	}

	if len(attestation.AttesterBitField) != (len(attestationIndices.Committee)+7)/8 {
		return fmt.Errorf("attestation bitfield length does not match number of validators in committee")
	}

	trailingZeros := uint8(len(attestationIndices.Committee) % 8)
	if !checkTrailingZeros(attestation.AttesterBitField[len(attestation.AttesterBitField)-1], trailingZeros) {
		return fmt.Errorf("expected %d bits at the end empty", trailingZeros)
	}

	var pubkey *bls.PublicKey
	for bit := 0; bit < len(attestationIndices.Committee); bit++ {
		set := (attestation.AttesterBitField[bit/8]>>uint(7-(bit%8)))%2 == 1
		if set {
			if pubkey == nil {
				pubkey = b.state.Crystallized.Validators[attestationIndices.Committee[bit]].Pubkey
			} else {
				pubkey, err = bls.AggregatePubKeys([]*bls.PublicKey{pubkey, b.state.Crystallized.Validators[attestationIndices.Committee[bit]].Pubkey})
			}
		}
	}

	forkVersion := b.state.Crystallized.PreForkVersion
	if attestation.Slot >= b.state.Crystallized.ForkSlotNumber {
		forkVersion = b.state.Crystallized.PostForkVersion
	}

	asd := transaction.AttestationSignedData{
		Version:        forkVersion,
		Slot:           attestation.Slot,
		Shard:          attestation.ShardID,
		ParentHashes:   hashes,
		ShardBlockHash: attestation.ShardBlockHash,
		JustifiedSlot:  attestation.JustifiedSlot,
	}

	bs := asd.Serialize()
	valid, err := bls.VerifySig(pubkey, bs, attestation.AggregateSignature)

	if err != nil || !valid {
		return errors.New("bls signature did not validate")
	}

	return nil
}

// AddBlock adds a block header to the current chain. The block should already
// have been validated by this point.
func (b *Blockchain) AddBlock(block *primitives.Block) error {
	logger.Debug("adding block to cache and updating head if needed", "hash", block.Hash())
	b.UpdateChainHead(block)

	b.db.SetBlock(block)

	return nil
}

// ProcessBlock is called when a block is received from a peer.
func (b Blockchain) ProcessBlock(block *primitives.Block) error {

	b.ApplyBlock(block)

	err := b.AddBlock(block)
	if err != nil {
		return err
	}

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

// getShardsAndCommitteesForSlot gets the committee for each shard.
func (b *Blockchain) getShardsAndCommitteesForSlot(slot uint64) []primitives.ShardAndCommittee {
	earliestSlotInArray := int(b.state.Crystallized.LastStateRecalculation) - b.config.CycleLength
	if earliestSlotInArray < 0 {
		earliestSlotInArray = 0
	}
	return b.state.Crystallized.ShardAndCommitteeForSlots[slot-uint64(earliestSlotInArray)]
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

func (b *Blockchain) getTotalActiveValidatorBalance() uint64 {
	total := uint64(0)
	for _, v := range b.state.Crystallized.Validators {
		if v.Status == Active {
			total += v.Balance
		}
	}
	return total
}

// totalValidatingBalance is the sum of the balances of active validators.
func (c *CrystallizedState) totalValidatingBalance() uint64 {
	total := uint64(0)
	for _, v := range c.Validators {
		total += v.Balance
	}
	return total
}

// applyBlockActiveStateChanges applys state changes from the block
// to the blockchain's state.
func (b *Blockchain) applyBlockActiveStateChanges(newBlock *primitives.Block) error {
	if len(newBlock.AncestorHashes) != 32 {
		return errors.New("ancestorHashes improperly formed")
	}

	parentBlock, err := b.db.GetBlockForHash(newBlock.AncestorHashes[0])
	if err != nil {
		return err
	}

	newHashes := UpdateAncestorHashes(parentBlock.AncestorHashes, parentBlock.SlotNumber, parentBlock.Hash())
	for i := range newBlock.AncestorHashes {
		if newHashes[i] != newBlock.AncestorHashes[i] {
			return errors.New("ancestor hashes don't match expected value")
		}
	}

	for _, a := range newBlock.Attestations {
		err := b.ValidateAttestation(&a, parentBlock, b.config)
		if err != nil {
			return err
		}

		newCache, err := b.state.CalculateNewVoteCache(newBlock, b.voteCache, b.config)
		if err != nil {
			return err
		}

		b.voteCache = newCache

		b.state.Active.PendingAttestations = append(b.state.Active.PendingAttestations, a)
	}
	for _, tx := range newBlock.Specials {
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

	shardAndCommittee := b.getShardsAndCommitteesForSlot(parentBlock.SlotNumber)[0]
	proposerIndex := int(parentBlock.SlotNumber % uint64(len(shardAndCommittee.Committee)))

	// validate parent block proposer
	if newBlock.SlotNumber != 0 {
		attestations := []transaction.Attestation{}
		for _, a := range newBlock.Attestations {
			attestations = append(attestations, a)
		}
		if len(attestations) == 0 {
			return errors.New("invalid parent block proposer")
		}

		attestation := attestations[0]
		if attestation.ShardID != shardAndCommittee.ShardID || attestation.Slot != parentBlock.SlotNumber || !hasVoted(attestation.AttesterBitField, proposerIndex) {
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

	tx := transaction.Transaction{Data: transaction.RandaoChangeTransaction{ProposerIndex: uint32(proposerIndex), NewRandao: newBlock.RandaoReveal}}

	b.state.Active.PendingActions = append(b.state.Active.PendingActions, tx)

	return nil
}

// addValidatorSetChangeRecord adds a validator addition/removal to the
// validator set hash change.
func (c *CrystallizedState) addValidatorSetChangeRecord(index uint32, pubkey []byte, flag uint8) {
	var indexBytes [4]byte
	binary.BigEndian.PutUint32(indexBytes[:], index)
	c.ValidatorSetDeltaHashChange = chainhash.HashH(serialization.AppendAll(c.ValidatorSetDeltaHashChange[:], indexBytes[:], pubkey, []byte{flag}))
}

// exitValidator exits a validator from being active to either
// penalized or pending an exit.
func (c *CrystallizedState) exitValidator(index uint32, penalize bool, currentSlot uint64, con *Config) {
	validator := &c.Validators[index]
	validator.ExitSlot = currentSlot
	if penalize {
		validator.Status = Penalized
		c.DepositsPenalizedInPeriod[currentSlot/con.WithdrawalPeriod] += validator.Balance
	} else {
		validator.Status = PendingExit
	}
}

// removeProcessedAttestations removes attestations from the list
// with a slot greater than the last state recalculation.
func removeProcessedAttestations(attestations []transaction.Attestation, lastStateRecalculation uint64) []transaction.Attestation {
	attestationsf := make([]transaction.Attestation, 0)
	for _, a := range attestations {
		if a.Slot > lastStateRecalculation {
			attestationsf = append(attestationsf, a)
		}
	}
	return attestationsf
}

// applyBlockCrystallizedStateChanges applies crystallized state changes up to
// a certain slot number.
func (b *Blockchain) applyBlockCrystallizedStateChanges(slotNumber uint64) error {
	// go through each cycle needed to get up to the specified slot number
	for slotNumber-b.state.Crystallized.LastStateRecalculation >= uint64(b.config.CycleLength) {
		// per-cycle parameters for reward calculation
		totalBalance := b.state.Crystallized.totalValidatingBalance()
		totalBalanceInCoins := totalBalance / UnitInCoin
		rewardQuotient := b.config.BaseRewardQuotient * uint64(math.Sqrt(float64(totalBalanceInCoins)))
		quadraticPenaltyQuotient := b.config.SqrtEDropTime * b.config.SqrtEDropTime
		timeSinceFinality := slotNumber - b.state.Crystallized.LastFinalizedSlot

		// go through each slot for that cycle
		for slot := b.state.Crystallized.LastStateRecalculation - uint64(b.config.CycleLength); slot < b.state.Crystallized.LastStateRecalculation-1; slot++ {
			shardsAndCommittees := b.getShardsAndCommitteesForSlot(slot)
			committee := shardsAndCommittees[0].Committee
			totalBalance := uint64(0)
			for _, i := range committee {
				totalBalance += b.state.Crystallized.Validators[i].Balance
			}
			attesterBalance := uint64(0)

			node, err := b.chain.GetBlock(int(slot))
			if err != nil {
				return err
			}
			block, err := b.db.GetBlockForHash(*node)
			if err != nil {
				return err
			}

			attestations := []transaction.Attestation{}
			for _, a := range block.Attestations {
				attestations = append(attestations, a)
			}
			if len(attestations) == 0 {
				return errors.New("invalid parent block proposer")
			}

			attestation := attestations[0]

			for index := range committee {
				if hasVoted(attestation.AttesterBitField, int(index)) {
					attesterBalance += b.state.Crystallized.Validators[index].Balance
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

			// adjust rewards
			for _, validatorID := range committee {
				if hasVoted(attestation.AttesterBitField, int(validatorID)) {
					if timeSinceFinality <= uint64(3*b.config.CycleLength) {
						balance := b.state.Crystallized.Validators[validatorID].Balance
						b.state.Crystallized.Validators[validatorID].Balance += balance / rewardQuotient * (2*attesterBalance - totalBalance)
					}
				} else {
					if timeSinceFinality <= uint64(3*b.config.CycleLength) {
						b.state.Crystallized.Validators[validatorID].Balance -= b.state.Crystallized.Validators[validatorID].Balance / rewardQuotient
					} else {
						balance := b.state.Crystallized.Validators[validatorID].Balance
						b.state.Crystallized.Validators[validatorID].Balance -= balance/rewardQuotient + balance*timeSinceFinality/quadraticPenaltyQuotient
					}
				}

			}

			if b.state.Crystallized.JustifiedStreak >= uint64(b.config.CycleLength+1) {
				if b.state.Crystallized.LastFinalizedSlot < slot-uint64(b.config.CycleLength-1) {
					b.state.Crystallized.LastFinalizedSlot = slot - uint64(b.config.CycleLength-1)
				}
			}

			shardProposals := make(map[chainhash.Hash]uint32)

			// populate shard proposals
			for _, a := range attestations {
				if _, success := shardProposals[a.ShardBlockHash]; !success {
					shardProposals[a.ShardBlockHash] = uint32(a.ShardID)
				}
			}

			for shardBlockHash, shard := range shardProposals {
				totalBalanceAttesting := b.voteCache[shardBlockHash].totalDeposit
				shardCommittee, err := ShardCommitteeByShardID(shard, shardsAndCommittees)
				if err != nil {
					return err
				}
				totalCommitteeBalance := uint64(0)

				// tally up the balance of each validator who voted for this hash
				for _, validatorIndex := range shardCommittee {
					if hasVoted(attestation.AttesterBitField, int(validatorIndex)) {
						totalCommitteeBalance += b.state.Crystallized.Validators[validatorIndex].Balance
					}
				}

				// if this is a super-majority, set up a cross-link
				if 3*totalCommitteeBalance >= 2*totalBalanceAttesting && !b.state.Crystallized.Crosslinks[shard].RecentlyChanged {
					timeSinceLastConfirmations := block.SlotNumber - b.state.Crystallized.Crosslinks[shard].Slot

					for _, validatorIndex := range shardCommittee {
						if !b.state.Crystallized.Crosslinks[shard].RecentlyChanged {
							checkBit := hasVoted(attestation.AttesterBitField, int(validatorIndex))
							if checkBit {
								balance := b.state.Crystallized.Validators[validatorIndex].Balance
								b.state.Crystallized.Validators[validatorIndex].Balance += balance / rewardQuotient * (2*totalBalanceAttesting - totalCommitteeBalance) / totalCommitteeBalance
							} else {
								balance := b.state.Crystallized.Validators[validatorIndex].Balance
								b.state.Crystallized.Validators[validatorIndex].Balance += balance/rewardQuotient + balance*timeSinceLastConfirmations/quadraticPenaltyQuotient
							}
						}
					}
					b.state.Crystallized.Crosslinks[shard] = primitives.Crosslink{
						RecentlyChanged: true,
						Slot:            b.state.Crystallized.LastStateRecalculation + uint64(b.config.CycleLength),
						Hash:            &shardBlockHash,
					}

				}
			}
		}

		for i := range b.state.Crystallized.Validators {
			if b.state.Crystallized.Validators[i].Status == Penalized {
				balance := b.state.Crystallized.Validators[i].Balance
				b.state.Crystallized.Validators[i].Balance -= balance/rewardQuotient + balance*timeSinceFinality/quadraticPenaltyQuotient
			}
		}

		for _, a := range b.state.Active.PendingActions {
			if t, success := a.Data.(transaction.LogoutTransaction); success {
				verified, err := bls.VerifySig(b.state.Crystallized.Validators[t.From].Pubkey, []byte("LOGOUT"), &t.Signature)
				if err != nil || !verified {
					// verification failed
					continue
				}
				if b.state.Crystallized.Validators[t.From].Status != Active {
					// can only log out from an active state
					continue
				}

				b.state.Crystallized.exitValidator(t.From, false, slotNumber, b.config)
			}
			if t, success := a.Data.(transaction.CasperSlashingTransaction); success {
				// TODO: verify signatures 1 + 2
				if bytes.Equal(t.SourceDataSigned, t.DestinationDataSigned) {
					// data must be distinct
					continue
				}

				validators := make(map[uint32]bool)
				validatorsInBoth := []uint32{}

				sourceBuf := bytes.NewBuffer(t.SourceDataSigned)
				destBuf := bytes.NewBuffer(t.DestinationDataSigned)

				var attestationTransactionSource transaction.Attestation
				var attestationTransactionDest transaction.Attestation

				err := binary.Read(destBuf, binary.BigEndian, attestationTransactionDest)
				if err != nil {
					continue
				}

				err = binary.Read(sourceBuf, binary.BigEndian, attestationTransactionSource)
				if err != nil {
					continue
				}

				for _, v := range t.DestinationValidators {
					validators[v] = true
				}
				for _, v := range t.SourceValidators {
					if _, success := validators[v]; success {
						validatorsInBoth = append(validatorsInBoth, v)
					}
				}

				for _, v := range validatorsInBoth {
					if b.state.Crystallized.Validators[v].Status != Penalized {
						b.state.Crystallized.exitValidator(v, true, slotNumber, b.config)
					}
				}
			}
			if t, success := a.Data.(transaction.RandaoRevealTransaction); success {
				b.state.Crystallized.Validators[t.ValidatorIndex].RandaoCommitment = t.Commitment
			}
		}

		for i, v := range b.state.Crystallized.Validators {
			if v.Status == Active && v.Balance < b.config.MinimumDepositSize {
				b.state.Crystallized.exitValidator(uint32(i), false, slotNumber, b.config)
			}
		}

		b.state.Crystallized.LastStateRecalculation += uint64(b.config.CycleLength)

		b.state.Active.PendingAttestations = removeProcessedAttestations(b.state.Active.PendingAttestations, b.state.Crystallized.LastStateRecalculation)

		b.state.Active.PendingActions = []transaction.Transaction{}

		b.state.Active.RecentBlockHashes = b.state.Active.RecentBlockHashes[b.config.CycleLength:]

		for i := 0; i < b.config.CycleLength; i++ {
			b.state.Crystallized.ShardAndCommitteeForSlots[i] = b.state.Crystallized.ShardAndCommitteeForSlots[b.config.CycleLength+i]
		}
	}
	return nil
}

const (
	// ValidatorEntry is a flag for validator set change meaning a validator was added to the set
	ValidatorEntry = iota

	// ValidatorExit is a flag for validator set change meaning a validator was removed from the set
	ValidatorExit
)

// ChangeValidatorSet updates the current validator set.
func (b *Blockchain) ChangeValidatorSet(validators []primitives.Validator, currentSlot uint64) error {
	activeValidators := GetActiveValidatorIndices(validators)

	totalBalance := uint64(0)
	for _, v := range activeValidators {
		totalBalance += b.state.Crystallized.Validators[v].Balance
	}

	maxAllowableChange := 2 * b.config.DepositSize * UnitInCoin
	if maxAllowableChange < totalBalance/b.config.MaxValidatorChurnQuotient {
		maxAllowableChange = totalBalance / b.config.MaxValidatorChurnQuotient
	}

	totalChanged := uint64(0)
	for i := range validators {
		if validators[i].Status == PendingActivation {
			validators[i].Status = Active
			totalChanged += b.config.DepositSize * UnitInCoin
			b.state.Crystallized.addValidatorSetChangeRecord(uint32(i), validators[i].Pubkey.Hash(), ValidatorEntry)
		}
		if validators[i].Status == PendingExit {
			validators[i].Status = PendingWithdraw
			validators[i].ExitSlot = currentSlot
			totalChanged += validators[i].Balance
			b.state.Crystallized.addValidatorSetChangeRecord(uint32(i), validators[i].Pubkey.Hash(), ValidatorExit)
		}
		if totalChanged >= maxAllowableChange {
			break
		}
	}

	periodIndex := currentSlot / b.config.WithdrawalPeriod
	totalPenalties := b.state.Crystallized.DepositsPenalizedInPeriod[periodIndex]
	if periodIndex >= 1 {
		totalPenalties += b.state.Crystallized.DepositsPenalizedInPeriod[periodIndex-1]
	}
	if periodIndex >= 2 {
		totalPenalties += b.state.Crystallized.DepositsPenalizedInPeriod[periodIndex-2]
	}

	for i := range validators {
		if (validators[i].Status == PendingWithdraw || validators[i].Status == Penalized) && currentSlot >= validators[i].ExitSlot+b.config.WithdrawalPeriod {
			if validators[i].Status == Penalized {
				validatorBalanceFactor := totalPenalties * 3 / totalBalance
				if totalBalance < 1 {
					validatorBalanceFactor = 1
				}
				validators[i].Balance -= validators[i].Balance * validatorBalanceFactor
			}
			validators[i].Status = Withdrawn

			// withdraw validators[i].balance to shard chain
		}
	}

	b.state.Crystallized.ValidatorSetChangeSlot = b.state.Crystallized.LastStateRecalculation
	for i := range b.state.Crystallized.Crosslinks {
		b.state.Crystallized.Crosslinks[i].RecentlyChanged = false
	}
	lastShardAndCommittee := b.state.Crystallized.ShardAndCommitteeForSlots[len(b.state.Crystallized.ShardAndCommitteeForSlots)-1]
	nextStartShard := (lastShardAndCommittee[len(lastShardAndCommittee)-1].ShardID + 1) % uint32(b.config.ShardCount)
	slotsForNextCycle := GetNewShuffling(b.state.Active.RandaoMix, validators, int(nextStartShard), b.config)

	for i := range slotsForNextCycle {
		b.state.Crystallized.ShardAndCommitteeForSlots[b.config.CycleLength+i] = slotsForNextCycle[i]
	}

	return nil
}

// ApplyBlock applies a block to the state
func (b *Blockchain) ApplyBlock(newBlock *primitives.Block) error {
	b.stateLock.Lock()
	defer b.stateLock.Unlock()
	err := b.applyBlockCrystallizedStateChanges(newBlock.SlotNumber)
	if err != nil {
		return err
	}

	err = b.applyBlockActiveStateChanges(newBlock)
	if err != nil {
		return err
	}

	// validator set change
	shouldChangeValidatorSet := true
	if newBlock.SlotNumber-b.state.Crystallized.ValidatorSetChangeSlot < b.config.MinimumValidatorSetChangeInterval {
		shouldChangeValidatorSet = false
	}
	if b.state.Crystallized.LastFinalizedSlot <= b.state.Crystallized.ValidatorSetChangeSlot {
		shouldChangeValidatorSet = false
	}
	for i := range b.state.Crystallized.ShardAndCommitteeForSlots {
		if b.state.Crystallized.Crosslinks[i].Slot <= b.state.Crystallized.ValidatorSetChangeSlot {
			shouldChangeValidatorSet = false
		}
	}

	if shouldChangeValidatorSet {
		err := b.ChangeValidatorSet(b.state.Crystallized.Validators, newBlock.SlotNumber)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetState gets a copy of the current state of the blockchain.
func (b *Blockchain) GetState() State {
	b.stateLock.Lock()
	state := b.state
	b.stateLock.Unlock()
	return state
}
