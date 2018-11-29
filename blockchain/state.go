package blockchain

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/golang/protobuf/proto"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/serialization"
	"github.com/phoreproject/synapse/transaction"
	logger "github.com/sirupsen/logrus"
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

// BeaconState is the state of a beacon block
type BeaconState struct {
	// Slot of last validator set change
	ValidatorSetChangeSlot uint64
	// List of validators
	Validators []primitives.Validator
	// Most recent crosslink for each shard
	Crosslinks []primitives.Crosslink
	// Last cycle-boundary state recalculation
	// MEMO: old name in our code is LastStateRecalculationSlot
	LastStateRecalculationSlot uint64
	// Last finalized slot
	LastFinalizedSlot uint64
	// Justification source
	JustificationSource          uint64
	PrevCycleJustificationSource uint64
	// Recent justified slot bitmask
	JustifiedSlotBitfield uint64
	// Committee members and their assigned shard, per slot
	ShardAndCommitteeForSlots [][]primitives.ShardAndCommittee
	// Persistent shard committees
	PersistentCommittees             []uint32
	PersistentCommitteeReassignments []ShardReassignmentRecord
	// Randao seed used for next shuffling
	NextShufflingSeed chainhash.Hash
	// Total deposits penalized in the given withdrawal period
	DepositsPenalizedInPeriod []uint64
	// Hash chain of validator set changes (for light clients to easily track deltas)
	ValidatorSetDeltaHashChange chainhash.Hash
	// Current sequence number for withdrawals
	CurrentExitSeq uint64
	// Genesis time
	GenesisTime uint64
	// Parameters relevant to hard forks / versioning.
	// Should be updated only by hard forks.
	ForkData ForkData
	// Attestations not yet processed
	//PendingAttestations []transaction.Attestation
	PendingAttestations []transaction.ProcessedAttestation
	RecentBlockHashes   []chainhash.Hash
	RandaoMix           chainhash.Hash
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

// CandidatePoWReceiptRootRecord is the record of candidate PoW receipt root
type CandidatePoWReceiptRootRecord struct {
	// Candidate PoW receipt root
	CandidatePoWReceiptRoot chainhash.Hash
	// Vote count
	Votes uint64
}

// ShardCommitteeByShardID gets the shards committee from a list of committees/shards
// in a list.
func ShardCommitteeByShardID(shardID uint64, shardCommittees []primitives.ShardAndCommittee) ([]uint32, error) {
	for _, s := range shardCommittees {
		if s.Shard == shardID {
			return s.Committee, nil
		}
	}

	return nil, fmt.Errorf("unable to find committee based on shard: %v", shardID)
}

// CommitteeInShardAndSlot gets the committee of validator indices at a specific
// shard and slot given the relative slot number [0, CYCLE_LENGTH] and shard ID.
func CommitteeInShardAndSlot(slotIndex uint64, shardID uint64, shardCommittees [][]primitives.ShardAndCommittee) ([]uint32, error) {
	shardCommittee := shardCommittees[slotIndex]

	return ShardCommitteeByShardID(shardID, shardCommittee)
}

func (s *BeaconState) updatePendingActions(newBlock *primitives.Block) {
	/*
		for _, tx := range newBlock.Specials {
			if _, success := tx.Data.(transaction.LoginTransaction); success {
				a.PendingActions = append(a.PendingActions, tx)
			}
			if _, success := tx.Data.(transaction.LogoutTransaction); success {
				a.PendingActions = append(a.PendingActions, tx)
			}
			if _, success := tx.Data.(transaction.RegisterTransaction); success {
				a.PendingActions = append(a.PendingActions, tx)
			}
		}
	*/
}

// GetAttesterIndices gets all of the validator indices involved with the committee
// assigned to the shard and slot of the committee.
func (s *BeaconState) GetAttesterIndices(slot uint64, shard uint64, con *Config) ([]uint32, error) {
	slotsStart := s.LastStateRecalculationSlot - uint64(con.CycleLength)
	slotIndex := (slot - slotsStart) % uint64(con.CycleLength)
	return CommitteeInShardAndSlot(slotIndex, shard, s.ShardAndCommitteeForSlots)
}

// GetAttesterCommitteeSize gets the size of committee
func (s *BeaconState) GetAttesterCommitteeSize(slot uint64, con *Config) uint32 {
	slotsStart := s.LastStateRecalculationSlot - uint64(con.CycleLength)
	slotIndex := (slot - slotsStart) % uint64(con.CycleLength)
	return uint32(len(s.ShardAndCommitteeForSlots[slotIndex]))
}

// GetCommitteeIndices gets all of the validator indices involved with the committee
// assigned to the shard and slot of the committee.
func (s *BeaconState) GetCommitteeIndices(slot uint64, shardID uint64, con *Config) ([]uint32, error) {
	slotsStart := s.LastStateRecalculationSlot - uint64(con.CycleLength)
	slotIndex := (slot - slotsStart) % uint64(con.CycleLength)
	return CommitteeInShardAndSlot(slotIndex, shardID, s.ShardAndCommitteeForSlots)
}

// InitializeState initializes state to the genesis state according to the config.
func (b *Blockchain) InitializeState(initialValidators []InitialValidatorEntry) error {
	b.stateLock.Lock()
	validators := make([]primitives.Validator, len(initialValidators))
	for i, v := range initialValidators {
		validators[i] = primitives.Validator{
			Pubkey:                v.PubKey,
			WithdrawalCredentials: v.WithdrawalCredentials,
			RandaoCommitment:      v.RandaoCommitment,
			Balance:               b.config.DepositSize,
			Status:                Active,
			ExitSeq:               0,
		}
	}

	x := GetNewShuffling(chainhash.HashH([]byte("nothing?")), validators, 0, b.config)

	crosslinks := make([]primitives.Crosslink, b.config.ShardCount)

	for i := 0; i < b.config.ShardCount; i++ {
		crosslinks[i] = primitives.Crosslink{
			Slot:           0,
			ShardBlockHash: zeroHash,
		}
	}

	recentBlockHashes := make([]chainhash.Hash, b.config.CycleLength*2)
	for i := 0; i < b.config.CycleLength*2; i++ {
		recentBlockHashes[i] = zeroHash
	}

	b.state = BeaconState{
		Validators:                  validators,
		ValidatorSetChangeSlot:      0,
		Crosslinks:                  crosslinks,
		LastStateRecalculationSlot:  0,
		LastFinalizedSlot:           0,
		JustificationSource:         0,
		JustifiedSlotBitfield:       0,
		ShardAndCommitteeForSlots:   append(x, x...),
		DepositsPenalizedInPeriod:   []uint64{0},
		ValidatorSetDeltaHashChange: zeroHash,
		ForkData: ForkData{
			PreForkVersion:  InitialForkVersion,
			PostForkVersion: InitialForkVersion,
			ForkSlotNumber:  0,
		},
		PendingAttestations: []transaction.ProcessedAttestation{},
		RecentBlockHashes:   recentBlockHashes,
		RandaoMix:           zeroHash,
	}

	ancestorHashes := make([]chainhash.Hash, 32)
	for i := range ancestorHashes {
		ancestorHashes[i] = zeroHash
	}

	block0 := primitives.Block{
		SlotNumber:            0,
		RandaoReveal:          chainhash.Hash{},
		AncestorHashes:        ancestorHashes,
		ActiveStateRoot:       zeroHash,
		CrystallizedStateRoot: zeroHash,
		Specials:              []transaction.Transaction{},
		Attestations:          []transaction.AttestationRecord{},
	}

	b.stateLock.Unlock()

	err := b.AddBlock(&block0)
	if err != nil {
		return err
	}

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
		return nil, 0, errors.New("validator proof of possession does not verify")
	}

	rec := primitives.Validator{
		Pubkey:                pubkey,
		WithdrawalCredentials: withdrawalAddress,
		RandaoCommitment:      randaoCommitment,
		Balance:               con.DepositSize,
		Status:                status,
		ExitSeq:               0,
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
		validatorsPerShard := Split(slotIndices, uint32(committeesPerSlot))

		shardIDStart := crosslinkingStart + slot*committeesPerSlot

		shardCommittees := make([]primitives.ShardAndCommittee, len(validatorsPerShard))
		for shardPosition, indices := range validatorsPerShard {
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

func validateAttestationSlot(attestation *transaction.AttestationRecord, parentBlock *primitives.Block, c *Config) error {
	if attestation.Data.Slot > parentBlock.SlotNumber {
		return errors.New("attestation slot number too high")
	}

	// verify attestation slot >= max(parent.slot - CYCLE_LENGTH + 1, 0)
	if attestation.Data.Slot < uint64(math.Max(float64(int64(parentBlock.SlotNumber)-int64(c.CycleLength)+1), 0)) {
		return errors.New("attestation slot number too low")
	}

	return nil
}

func (b *Blockchain) validateAttestationJustifiedBlock(attestation *transaction.AttestationRecord, parentBlock *primitives.Block, c *Config) error {
	if attestation.Data.JustifiedSlot > b.state.JustificationSource {
		return errors.New("last justified slot should be less than or equal to the crystallized slot")
	}

	justifiedBlockHash, err := b.chain.GetBlock(int(attestation.Data.JustifiedSlot))
	if err != nil {
		return errors.New("justified block not in index")
	}

	if !justifiedBlockHash.IsEqual(&attestation.Data.ShardBlockHash) {
		return errors.New("justified block hash does not match block at slot")
	}

	return nil
}

func (b *Blockchain) findAttestationPublicKey(attestation *transaction.AttestationRecord, parentBlock *primitives.Block, c *Config) (*bls.PublicKey, error) {
	attestationIndicesForShards := b.GetShardsAndCommitteesForSlot(attestation.Data.Slot)
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

	if len(attestation.AttesterBitfield) != (len(attestationIndices.Committee)+7)/8 {
		return nil, fmt.Errorf("attestation bitfield length does not match number of validators in committee")
	}

	trailingZeros := 8 - uint8(len(attestationIndices.Committee)%8)
	if trailingZeros != 8 && !checkTrailingZeros(attestation.AttesterBitfield[len(attestation.AttesterBitfield)-1], trailingZeros) {
		return nil, fmt.Errorf("expected %d bits at the end empty", trailingZeros)
	}

	var pubkey bls.PublicKey
	pubkeySet := false
	for bit := 0; bit < len(attestationIndices.Committee); bit++ {
		set := (attestation.AttesterBitfield[bit/8]>>uint(7-(bit%8)))%2 == 1
		if set {
			if !pubkeySet {
				pubkey = b.state.Validators[attestationIndices.Committee[bit]].Pubkey
				pubkeySet = true
			} else {
				p, err := bls.AggregatePubKeys([]*bls.PublicKey{&pubkey, &b.state.Validators[attestationIndices.Committee[bit]].Pubkey})
				if err != nil {
					return nil, err
				}
				pubkey = *p
			}
		}
	}

	return &pubkey, nil
}

func (b *Blockchain) validateAttestationSignature(attestation *transaction.AttestationRecord, parentBlock *primitives.Block, c *Config) error {
	pubkey, err := b.findAttestationPublicKey(attestation, parentBlock, c)
	if err != nil {
		return err
	}

	//forkVersion := b.state.PreForkVersion
	//if attestation.Slot >= b.state.ForkSlotNumber {
	//forkVersion = b.state.PostForkVersion
	//}

	obliqueParentHashesLen := len(attestation.Data.ParentHashes)
	hashes := make([]chainhash.Hash, b.config.CycleLength-obliqueParentHashesLen)

	for i := 1; i < b.config.CycleLength-obliqueParentHashesLen+1; i++ {
		h, err := b.chain.GetBlock(int(attestation.Data.Slot) - b.config.CycleLength + i)
		if err != nil {
			continue
		}

		hashes[i-1] = *h
	}

	asd := transaction.AttestationSignedData{
		Slot:           attestation.Data.Slot,
		Shard:          attestation.Data.Shard,
		ParentHashes:   hashes,
		ShardBlockHash: attestation.Data.ShardBlockHash,
		JustifiedSlot:  attestation.Data.JustifiedSlot,
	}

	bs, err := proto.Marshal(asd.ToProto())
	if err != nil {
		return err
	}
	valid, err := bls.VerifySig(pubkey, bs, &attestation.AggregateSig)

	if err != nil || !valid {
		return errors.New("bls signature did not validate")
	}

	return nil
}

// ValidateAttestationRecord checks attestation invariants and the BLS signature.
func (b *Blockchain) ValidateAttestationRecord(attestation *transaction.AttestationRecord, parentBlock *primitives.Block, c *Config) error {
	err := validateAttestationSlot(attestation, parentBlock, c)
	if err != nil {
		return err
	}

	err = b.validateAttestationJustifiedBlock(attestation, parentBlock, c)
	if err != nil {
		return err
	}

	err = b.validateAttestationSignature(attestation, parentBlock, c)
	if err != nil {
		return err
	}

	return nil
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

// UpdateAncestorHashes fills in the parent hash in ancestor hashes
// where the ith element represents the 2**i past block.
func UpdateAncestorHashes(parentAncestorHashes []chainhash.Hash, parentSlotNumber uint64, parentHash chainhash.Hash) []chainhash.Hash {
	newAncestorHashes := make([]chainhash.Hash, len(parentAncestorHashes))
	for i := uint(0); i < 32; i++ {
		if parentSlotNumber%(1<<i) == 0 {
			copy(newAncestorHashes[i][:], parentHash[:])
		} else {
			copy(newAncestorHashes[i][:], parentAncestorHashes[i][:])
		}
	}
	return newAncestorHashes
}

// GetShardsAndCommitteesForSlot gets the committee for each shard.
func (b *Blockchain) GetShardsAndCommitteesForSlot(slot uint64) []primitives.ShardAndCommittee {
	earliestSlotInArray := int(b.state.LastStateRecalculationSlot) - b.config.CycleLength
	if earliestSlotInArray < 0 {
		earliestSlotInArray = 0
	}
	return b.state.ShardAndCommitteeForSlots[slot-uint64(earliestSlotInArray)]
}

func hasVoted(bitfield []byte, index int) bool {
	return bitfield[index/8]&(128>>uint(index%8)) != 0
}

// RepeatHash repeats a hash n times.
func RepeatHash(h chainhash.Hash, n int) chainhash.Hash {
	for n > 0 {
		h = chainhash.HashH(h[:])
		n--
	}
	return h
}

// totalValidatingBalance is the sum of the balances of active validators.
func (s *BeaconState) totalValidatingBalance() uint64 {
	total := uint64(0)
	for _, v := range s.Validators {
		total += v.Balance
	}
	return total
}

func validateBlockAncestorHashes(newBlock, parentBlock *primitives.Block) error {
	if len(newBlock.AncestorHashes) != 32 {
		return errors.New("ancestorHashes improperly formed")
	}

	newHashes := UpdateAncestorHashes(parentBlock.AncestorHashes, parentBlock.SlotNumber, parentBlock.Hash())
	for i := range newBlock.AncestorHashes {
		if newHashes[i] != newBlock.AncestorHashes[i] {
			return errors.New("ancestor hashes don't match expected value")
		}
	}

	return nil
}

// applyBlockActiveStateChanges applys state changes from the block
// to the blockchain's state.
func (b *Blockchain) applyBlockActiveStateChanges(newBlock *primitives.Block) error {
	parentBlock, err := b.db.GetBlockForHash(newBlock.AncestorHashes[0])
	if err != nil {
		return err
	}

	err = validateBlockAncestorHashes(newBlock, parentBlock)
	if err != nil {
		return err
	}

	b.state.RecentBlockHashes = GetNewRecentBlockHashes(b.state.RecentBlockHashes, parentBlock.SlotNumber, newBlock.SlotNumber, parentBlock.Hash())

	err = b.state.CalculateNewVoteCache(newBlock, b.voteCache, b.config)
	if err != nil {
		return err
	}

	for _, a := range newBlock.Attestations {
		err := b.ValidateAttestationRecord(&a, parentBlock, b.config)
		if err != nil {
			return err
		}

		b.state.PendingAttestations = append(b.state.PendingAttestations, transaction.ProcessedAttestation{
			Data:             a.Data,
			AttesterBitfield: a.AttesterBitfield[:],
			PoCBitfield:      a.PoCBitfield,
			SlotIncluded:     newBlock.SlotNumber,
		})
	}

	b.state.updatePendingActions(newBlock)

	shardAndCommittee := b.GetShardsAndCommitteesForSlot(parentBlock.SlotNumber)[0]
	proposerIndex := int(parentBlock.SlotNumber % uint64(len(shardAndCommittee.Committee)))

	// validate parent block proposer
	if newBlock.SlotNumber != 0 {
		if len(newBlock.Attestations) == 0 {
			return errors.New("invalid parent block proposer")
		}

		attestation := newBlock.Attestations[0]
		if attestation.Data.Shard != shardAndCommittee.Shard || attestation.Data.Slot != parentBlock.SlotNumber || !hasVoted(attestation.AttesterBitfield, proposerIndex) {
			return errors.New("invalid parent block proposer")
		}
	}

	// TODO: fix tests with this
	// validator := b.state.Validators[proposerIndex]

	// expected := RepeatHash(newBlock.RandaoReveal, int((newBlock.SlotNumber-validator.RandaoLastChange)/uint64(b.config.RandaoSlotsPerLayer)+1))

	// if expected != validator.RandaoCommitment {
	// 	return errors.New("randao does not match commitment")
	// }

	for i := range b.state.RandaoMix {
		b.state.RandaoMix[i] ^= newBlock.RandaoReveal[i]
	}

	//tx := transaction.Transaction{Data: transaction.RandaoChangeTransaction{ProposerIndex: uint32(proposerIndex), NewRandao: b.state.RandaoMix}}
	//b.state.PendingActions = append(b.state.PendingActions, tx)

	return nil
}

// addValidatorSetChangeRecord adds a validator addition/removal to the
// validator set hash change.
func (s *BeaconState) addValidatorSetChangeRecord(index uint32, pubkey []byte, flag uint8) {
	var indexBytes [4]byte
	binary.BigEndian.PutUint32(indexBytes[:], index)
	s.ValidatorSetDeltaHashChange = chainhash.HashH(serialization.AppendAll(s.ValidatorSetDeltaHashChange[:], indexBytes[:], pubkey, []byte{flag}))
}

// exitValidator exits a validator from being active to either
// penalized or pending an exit.
func (s *BeaconState) exitValidator(index uint32, penalize bool, currentSlot uint64, con *Config) {
	validator := &s.Validators[index]
	validator.ExitSeq = currentSlot
	if penalize {
		validator.Status = Penalized
		if uint64(len(s.DepositsPenalizedInPeriod)) > currentSlot/con.WithdrawalPeriod {
			s.DepositsPenalizedInPeriod[currentSlot/con.WithdrawalPeriod] += validator.Balance
		} else {
			s.DepositsPenalizedInPeriod[currentSlot/con.WithdrawalPeriod] = validator.Balance
		}
	} else {
		validator.Status = PendingExit
	}
}

/*
func (s *BeaconState) resetCrosslinksRecentlyChanged() {
	for i := range s.Crosslinks {
		s.Crosslinks[i].RecentlyChanged = false
	}
}
*/

// removeProcessedAttestations removes attestations from the list
// with a slot greater than the last state recalculation.
func removeProcessedAttestations(attestations []transaction.ProcessedAttestation, lastStateRecalculation uint64) []transaction.ProcessedAttestation {
	attestationsf := make([]transaction.ProcessedAttestation, 0)
	for _, a := range attestations {
		if a.Data.Slot > lastStateRecalculation {
			attestationsf = append(attestationsf, a)
		}
	}
	return attestationsf
}

func (b *Blockchain) updateBlockCrystallizedStateChangesDataReward(totalBalance uint64, rewardQuotient uint64, quadraticPenaltyQuotient uint64, timeSinceFinality uint64) {
	lastStateRecalculationSlotCycleBack := uint64(0)
	if b.state.LastStateRecalculationSlot >= uint64(b.config.CycleLength) {
		lastStateRecalculationSlotCycleBack = b.state.LastStateRecalculationSlot - uint64(b.config.CycleLength)
	}

	activeValidators := GetActiveValidatorIndices(b.state.Validators)

	// go through each slot for that cycle
	for i := uint64(0); i < uint64(b.config.CycleLength); i++ {
		slot := i + lastStateRecalculationSlotCycleBack

		blockHash := b.state.RecentBlockHashes[i]

		voteCache := b.voteCache[blockHash]

		// TODO: need to change according to the spec
		// but currently there is obvious errors in the spec, i.e, in section Adjust justified slots and crosslink status, commit 7359b369642c4ee72e07a9745fc5dd253b7ca061
		// "If 3 * prev_cycle_boundary_attesting_balance >= 2 * total_balance then set state.justified_slot_bitfield &= 2 (ie. flip the second lowest bit to 1) and new_justification_source = s - CYCLE_LENGTH."
		// "state.justified_slot_bitfield &= 2" makes the bitfield either 2 or 0 and discard all other bits...
		// It should be "state.justified_slot_bitfield |= 2"
		//b.state.JustifiedSlotBitfield <<= 1
		if 3*voteCache.totalDeposit >= 2*totalBalance {
			if slot > b.state.JustificationSource {
				b.state.JustificationSource = slot
			}
			//b.state.JustificationSource &= 2
			b.state.JustificationSource++
		} else {
			b.state.JustificationSource = 0
		}

		if b.state.JustificationSource >= uint64(b.config.CycleLength+1) {
			if b.state.LastFinalizedSlot < slot-uint64(b.config.CycleLength-1) {
				b.state.LastFinalizedSlot = slot - uint64(b.config.CycleLength-1)
			}
		}

		// adjust rewards
		if timeSinceFinality <= uint64(3*b.config.CycleLength) {
			for _, validatorID := range activeValidators {
				_, voted := voteCache.validatorIndices[validatorID]
				if voted {
					balance := b.state.Validators[validatorID].Balance
					b.state.Validators[validatorID].Balance += balance / rewardQuotient * (2*voteCache.totalDeposit - totalBalance) / totalBalance
				} else {
					b.state.Validators[validatorID].Balance -= b.state.Validators[validatorID].Balance / rewardQuotient
				}
			}
		} else {
			for _, validatorID := range activeValidators {
				_, voted := voteCache.validatorIndices[validatorID]
				if !voted {
					balance := b.state.Validators[validatorID].Balance
					b.state.Validators[validatorID].Balance -= balance/rewardQuotient + balance*timeSinceFinality/quadraticPenaltyQuotient
				}
			}
		}

	}
}

func (b *Blockchain) updateBlockCrystallizedStateChangesDataPendingAttestations(slotNumber uint64, rewardQuotient uint64, quadraticPenaltyQuotient uint64) error {
	for _, a := range b.state.PendingAttestations {
		indices, err := b.state.GetAttesterIndices(a.Data.Slot, a.Data.Shard, b.config)
		if err != nil {
			return err
		}

		committeeSize := b.state.GetAttesterCommitteeSize(a.Data.Slot, b.config)

		totalVotingBalance := uint64(0)
		totalBalance := uint64(0)

		// tally up the balance of each validator who voted for this hash
		for _, validatorIndex := range indices {
			if hasVoted(a.AttesterBitfield, int(validatorIndex%committeeSize)) {
				totalVotingBalance += b.state.Validators[validatorIndex].Balance
			}
			totalBalance += b.state.Validators[validatorIndex].Balance
		}

		timeSinceLastConfirmations := slotNumber - b.state.Crosslinks[a.Data.Shard].Slot

		// if this is a super-majority, set up a cross-link
		if 3*totalVotingBalance >= 2*totalBalance /*&& !b.state.Crosslinks[a.ShardID].RecentlyChanged*/ {
			b.state.Crosslinks[a.Data.Shard] = primitives.Crosslink{
				//RecentlyChanged: true,
				Slot:           b.state.LastStateRecalculationSlot + uint64(b.config.CycleLength),
				ShardBlockHash: a.Data.ShardBlockHash,
			}
		}

		for _, validatorIndex := range indices {
			//if !b.state.Crosslinks[a.ShardID].RecentlyChanged {
			checkBit := hasVoted(a.AttesterBitfield, int(validatorIndex%committeeSize))
			if checkBit {
				balance := b.state.Validators[validatorIndex].Balance
				b.state.Validators[validatorIndex].Balance += balance / rewardQuotient * (2*totalVotingBalance - totalBalance) / totalBalance
			} else {
				balance := b.state.Validators[validatorIndex].Balance
				b.state.Validators[validatorIndex].Balance += balance/rewardQuotient + balance*timeSinceLastConfirmations/quadraticPenaltyQuotient
			}
			//}
		}
	}

	return nil
}

func (b *Blockchain) updateBlockCrystallizedStateChangesDataPenalizedValidators(rewardQuotient uint64, quadraticPenaltyQuotient uint64, timeSinceFinality uint64) {
	for i := range b.state.Validators {
		if b.state.Validators[i].Status == Penalized {
			balance := b.state.Validators[i].Balance
			b.state.Validators[i].Balance -= balance/rewardQuotient + balance*timeSinceFinality/quadraticPenaltyQuotient
		}
	}
}

func getValidatorsInBothSourceAndDestination(t transaction.CasperSlashingTransaction) []uint32 {
	validators := make(map[uint32]bool)
	validatorsInBoth := []uint32{}

	for _, v := range t.DestinationValidators {
		validators[v] = true
	}
	for _, v := range t.SourceValidators {
		if _, success := validators[v]; success {
			validatorsInBoth = append(validatorsInBoth, v)
		}
	}

	return validatorsInBoth
}

func (b *Blockchain) updateBlockCrystallizedStateChangesDataPendingActions(slotNumber uint64) {
	/*
		for _, a := range b.state.Active.PendingActions {
			if t, success := a.Data.(transaction.LogoutTransaction); success {
				verified, err := bls.VerifySig(&b.state.Crystallized.Validators[t.From].Pubkey, []byte("LOGOUT"), &t.Signature)
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
				if t, success := a.Data.(transaction.CasperSlashingTransaction); success {
					// TODO: verify signatures 1 + 2
					if bytes.Equal(t.SourceDataSigned, t.DestinationDataSigned) {
						// data must be distinct
						continue
					}

					sourceBuf := bytes.NewBuffer(t.SourceDataSigned)
					destBuf := bytes.NewBuffer(t.DestinationDataSigned)

					var attestationTransactionSource pb.Attestation
					var attestationTransactionDest pb.Attestation

					err := proto.Unmarshal(sourceBuf.Bytes(), &attestationTransactionSource)
					if err != nil {
						continue
					}

					err = proto.Unmarshal(destBuf.Bytes(), &attestationTransactionDest)
					if err != nil {
						continue
					}

					validatorsInBoth := getValidatorsInBothSourceAndDestination(t)

				for _, v := range validatorsInBoth {
					if b.state.Crystallized.Validators[v].Status != Penalized {
						b.state.Crystallized.exitValidator(v, true, slotNumber, b.config)
					}
				}
				if t, success := a.Data.(transaction.RandaoRevealTransaction); success {
					b.state.Validators[t.ValidatorIndex].RandaoCommitment = t.Commitment
				}
			}
	*/
}

func (b *Blockchain) updateBlockCrystallizedStateChangesDataInsufficientBalance(slotNumber uint64) {
	for i, v := range b.state.Validators {
		if v.Status == Active && v.Balance < b.config.MinimumDepositSize {
			b.state.exitValidator(uint32(i), false, slotNumber, b.config)
		}
	}
}

// applyBlockCrystallizedStateChanges applies crystallized state changes up to
// a certain slot number.
func (b *Blockchain) applyBlockCrystallizedStateChanges(slotNumber uint64) error {
	// go through each cycle needed to get up to the specified slot number
	for slotNumber-b.state.LastStateRecalculationSlot >= uint64(b.config.CycleLength) {
		totalBalance := b.state.totalValidatingBalance()
		totalBalanceInCoins := totalBalance / UnitInCoin
		rewardQuotient := b.config.BaseRewardQuotient * uint64(math.Sqrt(float64(totalBalanceInCoins)))
		quadraticPenaltyQuotient := b.config.SqrtEDropTime * b.config.SqrtEDropTime
		timeSinceFinality := slotNumber - b.state.LastFinalizedSlot

		b.updateBlockCrystallizedStateChangesDataReward(totalBalance, rewardQuotient, quadraticPenaltyQuotient, timeSinceFinality)

		err := b.updateBlockCrystallizedStateChangesDataPendingAttestations(slotNumber, rewardQuotient, quadraticPenaltyQuotient)
		if err != nil {
			return err
		}

		b.updateBlockCrystallizedStateChangesDataPenalizedValidators(rewardQuotient, quadraticPenaltyQuotient, timeSinceFinality)

		b.updateBlockCrystallizedStateChangesDataPendingActions(slotNumber)

		b.updateBlockCrystallizedStateChangesDataInsufficientBalance(slotNumber)

		b.state.LastStateRecalculationSlot += uint64(b.config.CycleLength)

		b.state.PendingAttestations = removeProcessedAttestations(b.state.PendingAttestations, b.state.LastStateRecalculationSlot)

		//b.state.PendingActions = []transaction.Transaction{}

		b.state.RecentBlockHashes = b.state.RecentBlockHashes[b.config.CycleLength:]

		for i := 0; i < b.config.CycleLength; i++ {
			b.state.ShardAndCommitteeForSlots[i] = b.state.ShardAndCommitteeForSlots[b.config.CycleLength+i]
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

func getActiveValidatorsTotalBalance(validators []primitives.Validator, state *BeaconState) uint64 {
	activeValidators := GetActiveValidatorIndices(validators)

	totalBalance := uint64(0)
	for _, v := range activeValidators {
		totalBalance += state.Validators[v].Balance
	}

	return totalBalance
}

// ChangeValidatorSet updates the current validator set.
func (b *Blockchain) ChangeValidatorSet(validators []primitives.Validator, currentSlot uint64) error {
	totalBalance := getActiveValidatorsTotalBalance(validators, &b.state)

	maxAllowableChange := 2 * b.config.DepositSize * UnitInCoin
	if maxAllowableChange < totalBalance/b.config.MaxValidatorChurnQuotient {
		maxAllowableChange = totalBalance / b.config.MaxValidatorChurnQuotient
	}

	totalChanged := uint64(0)
	for i := range validators {
		if validators[i].Status == PendingActivation {
			validators[i].Status = Active
			totalChanged += b.config.DepositSize * UnitInCoin
			b.state.addValidatorSetChangeRecord(uint32(i), validators[i].Pubkey.Hash(), ValidatorEntry)
		}
		if validators[i].Status == PendingExit {
			validators[i].Status = PendingWithdraw
			validators[i].ExitSeq = currentSlot
			totalChanged += validators[i].Balance
			b.state.addValidatorSetChangeRecord(uint32(i), validators[i].Pubkey.Hash(), ValidatorExit)
		}
		if totalChanged >= maxAllowableChange {
			break
		}
	}

	periodIndex := currentSlot / b.config.WithdrawalPeriod
	totalPenalties := b.state.DepositsPenalizedInPeriod[periodIndex]
	if periodIndex >= 1 {
		totalPenalties += b.state.DepositsPenalizedInPeriod[periodIndex-1]
	}
	if periodIndex >= 2 {
		totalPenalties += b.state.DepositsPenalizedInPeriod[periodIndex-2]
	}

	for i := range validators {
		if (validators[i].Status == PendingWithdraw || validators[i].Status == Penalized) && currentSlot >= validators[i].ExitSeq+b.config.WithdrawalPeriod {
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

	b.state.ValidatorSetChangeSlot = b.state.LastStateRecalculationSlot
	//b.state.resetCrosslinksRecentlyChanged()

	lastShardAndCommittee := b.state.ShardAndCommitteeForSlots[len(b.state.ShardAndCommitteeForSlots)-1]
	nextStartShard := (lastShardAndCommittee[len(lastShardAndCommittee)-1].Shard + 1) % uint64(b.config.ShardCount)
	slotsForNextCycle := GetNewShuffling(b.state.RandaoMix, validators, int(nextStartShard), b.config)

	for i := range slotsForNextCycle {
		b.state.ShardAndCommitteeForSlots[b.config.CycleLength+i] = slotsForNextCycle[i]
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
	if newBlock.SlotNumber-b.state.ValidatorSetChangeSlot < b.config.MinimumValidatorSetChangeInterval {
		shouldChangeValidatorSet = false
	}
	if b.state.LastFinalizedSlot <= b.state.ValidatorSetChangeSlot {
		shouldChangeValidatorSet = false
	}
	for _, slot := range b.state.ShardAndCommitteeForSlots {
		for _, committee := range slot {
			if b.state.Crosslinks[committee.Shard].Slot <= b.state.ValidatorSetChangeSlot {
				shouldChangeValidatorSet = false
			}
		}
	}

	if shouldChangeValidatorSet {
		err := b.ChangeValidatorSet(b.state.Validators, newBlock.SlotNumber)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetState gets a copy of the current state of the blockchain.
func (b *Blockchain) GetState() BeaconState {
	b.stateLock.Lock()
	state := b.state
	b.stateLock.Unlock()
	return state
}
