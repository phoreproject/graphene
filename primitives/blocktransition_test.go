package primitives_test

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/ssz"
	"github.com/phoreproject/synapse/validator"
	"github.com/sirupsen/logrus"
)

// SetupState initializes state with a certain number of initial validators
func SetupState(initialValidators int, c *config.Config) (*primitives.State, validator.Keystore, error) {
	keystore := validator.NewFakeKeyStore()

	var validators []primitives.InitialValidatorEntry

	for i := 0; i < initialValidators; i++ {
		key := keystore.GetKeyForValidator(uint32(i))
		pub := key.DerivePublicKey()
		hashPub, err := ssz.HashTreeRoot(pub.Serialize())
		if err != nil {
			return nil, nil, err
		}
		proofOfPossession, err := bls.Sign(key, hashPub[:], bls.DomainDeposit)
		if err != nil {
			return nil, nil, err
		}
		validators = append(validators, primitives.InitialValidatorEntry{
			PubKey:                pub.Serialize(),
			ProofOfPossession:     proofOfPossession.Serialize(),
			WithdrawalShard:       1,
			WithdrawalCredentials: chainhash.Hash{},
			DepositSize:           c.MaxDeposit,
		})
	}

	s, err := primitives.InitializeState(c, validators, uint64(time.Now().Unix()), false)
	return s, &keystore, err
}

// FakeBlockView always returns 0-hash for all slots and the last state root.
type FakeBlockView struct{}

func (f FakeBlockView) GetHashBySlot(slot uint64) (chainhash.Hash, error) {
	return chainhash.Hash{}, nil
}

func (f FakeBlockView) Tip() (chainhash.Hash, error) {
	return chainhash.Hash{}, nil
}

func (f FakeBlockView) SetTipSlot(slot uint64) {}

func (f FakeBlockView) GetLastStateRoot() (chainhash.Hash, error) {
	return chainhash.Hash{}, nil
}

func SignBlock(block *primitives.Block, key *bls.SecretKey, c *config.Config) error {
	var slotBytes [8]byte
	binary.BigEndian.PutUint64(slotBytes[:], block.BlockHeader.SlotNumber)
	slotBytesHash := chainhash.HashH(slotBytes[:])

	slotBytesSignature, err := bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		return err
	}

	block.BlockHeader.RandaoReveal = slotBytesSignature.Serialize()
	block.BlockHeader.Signature = bls.EmptySignature.Serialize()

	blockWithoutSignatureRoot, err := ssz.HashTreeRoot(block)
	if err != nil {
		return err
	}

	proposal := primitives.ProposalSignedData{
		Slot:      block.BlockHeader.SlotNumber,
		Shard:     c.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot, err := ssz.HashTreeRoot(proposal)
	if err != nil {
		return err
	}

	blockSignature, err := bls.Sign(key, proposalRoot[:], bls.DomainProposal)
	if err != nil {
		return err
	}

	block.BlockHeader.Signature = blockSignature.Serialize()

	return nil
}

func TestProposerIndex(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	slotToPropose := state.Slot + 1

	stateSlot := state.EpochIndex * c.EpochLength
	slotIndex := int64(slotToPropose-1) - int64(stateSlot) + int64(stateSlot%c.EpochLength) + int64(c.EpochLength)

	firstCommittee := state.ShardAndCommitteeForSlots[slotIndex][0].Committee

	proposerIndex := firstCommittee[int(slotToPropose-1)%len(firstCommittee)]

	blockTest := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessBlock(blockTest, c, FakeBlockView{}, true)
	if err != nil {
		t.Fatal(err)
	}

	slotToPropose = state.Slot + 1

	stateSlot = state.EpochIndex * c.EpochLength
	slotIndex = int64(slotToPropose-1) - int64(stateSlot) + int64(stateSlot%c.EpochLength) + int64(c.EpochLength)

	firstCommittee = state.ShardAndCommitteeForSlots[slotIndex][0].Committee

	proposerIndex = firstCommittee[int(slotToPropose-1)%len(firstCommittee)]

	blockTest = &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex-1), c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessBlock(blockTest, c, FakeBlockView{}, true)
	if err == nil {
		t.Fatal("expected block with wrong validator to fail")
	}
}

func TestBlockMaximums(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	slotToPropose := state.Slot + 1

	stateSlot := state.EpochIndex * c.EpochLength
	slotIndex := int64(slotToPropose-1) - int64(stateSlot) + int64(stateSlot%c.EpochLength) + int64(c.EpochLength)

	firstCommittee := state.ShardAndCommitteeForSlots[slotIndex][0].Committee

	proposerIndex := firstCommittee[int(slotToPropose-1)%len(firstCommittee)]

	type BodyShouldValidate struct {
		b         primitives.BlockBody
		validates bool
		reason    string
	}

	// the regtest config allows 1 of each transaction
	testBlockBodies := []BodyShouldValidate{
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			true,
			"should validate with maximum number for all",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}, {}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			false,
			"should not validate with too many attestations",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}, {}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			false,
			"should not validate with too many deposits",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}, {}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			false,
			"should not validate with too many exits",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}, {}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			false,
			"should not validate with too many proposer slashings",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}, {}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			false,
			"should not validate with too many CASPER slashings",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}, {}},
			},
			false,
			"should not validate with too many votes slashings",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{},
				Deposits:          []primitives.Deposit{},
				Exits:             []primitives.Exit{},
				ProposerSlashings: []primitives.ProposerSlashing{},
				CasperSlashings:   []primitives.CasperSlashing{},
				Votes:             []primitives.AggregatedVote{},
			},
			true,
			"should validate with empty block",
		},
	}

	for _, b := range testBlockBodies {
		blockTest := &primitives.Block{
			BlockHeader: primitives.BlockHeader{
				SlotNumber:     slotToPropose,
				ParentRoot:     chainhash.Hash{},
				StateRoot:      chainhash.Hash{},
				RandaoReveal:   [48]byte{},
				Signature:      [48]byte{},
				ValidatorIndex: proposerIndex,
			},
			BlockBody: b.b,
		}

		err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
		if err != nil {
			t.Fatal(err)
		}

		stateCopy := state.Copy()

		err = stateCopy.ProcessSlot(chainhash.Hash{}, c)
		if err != nil {
			t.Fatal(err)
		}

		failedBecauseOfMax := false

		err = stateCopy.ProcessBlock(blockTest, c, FakeBlockView{}, true)

		if err != nil {
			failedBecauseOfMax = strings.Contains(err.Error(), "maximum")
		}

		if failedBecauseOfMax == b.validates {
			if b.validates {
				t.Fatalf("expected block to validate, but got error: %s", err)
			} else {
				t.Fatalf("expected block to fail with reason: %s", b.reason)
			}
		}
	}
}

func TestProposerSlashingValidation(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	// sign two conflicting blocks at the same height
	slotToPropose := state.Slot + 1
	proposerIndex, err := state.GetBeaconProposerIndex(slotToPropose-1, c)
	if err != nil {
		t.Fatal(err)
	}

	type BodyShouldValidate struct {
		ps        primitives.ProposerSlashing
		validates bool
		reason    string
	}

	block1 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotToPropose,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: [48]byte{},
			Signature:    [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	key := keystore.GetKeyForValidator(proposerIndex)

	var slotBytes [8]byte
	binary.BigEndian.PutUint64(slotBytes[:], block1.BlockHeader.SlotNumber)
	slotBytesHash := chainhash.HashH(slotBytes[:])

	slotBytesSignature, err := bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		t.Fatal(err)
	}

	block1.BlockHeader.RandaoReveal = slotBytesSignature.Serialize()
	block1.BlockHeader.Signature = bls.EmptySignature.Serialize()

	blockWithoutSignatureRoot, err := ssz.HashTreeRoot(block1)
	if err != nil {
		t.Fatal(err)
	}

	proposal1 := primitives.ProposalSignedData{
		Slot:      block1.BlockHeader.SlotNumber,
		Shard:     c.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot1, err := ssz.HashTreeRoot(proposal1)
	if err != nil {
		t.Fatal(err)
	}

	blockSignature1, err := bls.Sign(key, proposalRoot1[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	block2 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotToPropose,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: [48]byte{},
			Signature:    [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations:      []primitives.Attestation{{}},
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	binary.BigEndian.PutUint64(slotBytes[:], block2.BlockHeader.SlotNumber)
	slotBytesHash = chainhash.HashH(slotBytes[:])

	slotBytesSignature, err = bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		t.Fatal(err)
	}

	block2.BlockHeader.RandaoReveal = slotBytesSignature.Serialize()
	block2.BlockHeader.Signature = bls.EmptySignature.Serialize()

	blockWithoutSignatureRoot, err = ssz.HashTreeRoot(block2)
	if err != nil {
		t.Fatal(err)
	}

	proposal2 := primitives.ProposalSignedData{
		Slot:      block2.BlockHeader.SlotNumber,
		Shard:     c.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot2, err := ssz.HashTreeRoot(proposal2)
	if err != nil {
		t.Fatal(err)
	}

	blockSignature2, err := bls.Sign(key, proposalRoot2[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	proposal2DifferentShard := proposal2.Copy()
	proposal2DifferentShard.Shard++

	proposal2DifferentShardHash, err := ssz.HashTreeRoot(proposal2DifferentShard)
	if err != nil {
		t.Fatal(err)
	}

	proposal2DifferentShardSignature, err := bls.Sign(key, proposal2DifferentShardHash[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	proposal2DifferentSlot := proposal2.Copy()
	proposal2DifferentSlot.Slot++

	proposal2DifferentSlotHash, err := ssz.HashTreeRoot(proposal2DifferentSlot)
	if err != nil {
		t.Fatal(err)
	}

	proposal2DifferentSlotSignature, err := bls.Sign(key, proposal2DifferentSlotHash[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	// ProposerSlashings happen when validators sign two blocks at the same slot.

	testBlockBodies := []BodyShouldValidate{
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal1,
				ProposalSignature1: blockSignature1.Serialize(),
				ProposalData2:      proposal2,
				ProposalSignature2: blockSignature2.Serialize(),
			},
			true,
			"should validate with two different signed blocks",
		},
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal1,
				ProposalSignature1: blockSignature1.Serialize(),
				ProposalData2:      proposal1,
				ProposalSignature2: blockSignature1.Serialize(),
			},
			false,
			"should not validate with same signed block",
		},
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal1,
				ProposalSignature1: [48]byte{},
				ProposalData2:      proposal2,
				ProposalSignature2: blockSignature2.Serialize(),
			},
			false,
			"should not validate with invalid signature 1",
		},
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal1,
				ProposalSignature1: blockSignature1.Serialize(),
				ProposalData2:      proposal2,
				ProposalSignature2: [48]byte{},
			},
			false,
			"should not validate with invalid signature 2",
		},
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal2,
				ProposalSignature1: blockSignature2.Serialize(),
				ProposalData2:      proposal2DifferentShard,
				ProposalSignature2: proposal2DifferentShardSignature.Serialize(),
			},
			false,
			"should not validate with different shard",
		},
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal2,
				ProposalSignature1: blockSignature2.Serialize(),
				ProposalData2:      proposal2DifferentSlot,
				ProposalSignature2: proposal2DifferentSlotSignature.Serialize(),
			},
			false,
			"should not validate with different slot",
		},
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	for _, b := range testBlockBodies {
		blockTest := &primitives.Block{
			BlockHeader: primitives.BlockHeader{
				SlotNumber:     slotToPropose,
				ParentRoot:     chainhash.Hash{},
				StateRoot:      chainhash.Hash{},
				RandaoReveal:   [48]byte{},
				Signature:      [48]byte{},
				ValidatorIndex: proposerIndex,
			},
			BlockBody: primitives.BlockBody{
				Attestations: nil,
				ProposerSlashings: []primitives.ProposerSlashing{
					b.ps,
				},
				CasperSlashings: nil,
				Deposits:        nil,
				Exits:           nil,
			},
		}

		err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
		if err != nil {
			t.Fatal(err)
		}

		stateCopy := state.Copy()

		err = stateCopy.ProcessBlock(blockTest, c, FakeBlockView{}, true)

		if (err == nil) != b.validates {
			if b.validates {
				t.Fatalf("expected block to validate, but got error: %s for case: %s", err, b.reason)
			} else {
				t.Fatalf("expected block to fail with reason: %s", b.reason)
			}
		}
	}
}

func TestVoteValidation(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	slotToPropose := state.Slot + 1
	proposerIndex, err := state.GetBeaconProposerIndex(slotToPropose-1, c)
	if err != nil {
		t.Fatal(err)
	}

	block1 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	key := keystore.GetKeyForValidator(proposerIndex)

	err = SignBlock(block1, key, c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	proposalVote := primitives.VoteData{
		Type:       primitives.Propose,
		Shards:     []uint32{1, 2},
		ActionHash: chainhash.Hash{},
		Proposer:   0,
	}

	h, _ := ssz.HashTreeRoot(proposalVote)

	signatureValidator1, err := bls.Sign(keystore.GetKeyForValidator(1), h[:], bls.DomainVote)
	if err != nil {
		t.Fatal(err)
	}
	signatureValidator0, err := bls.Sign(keystore.GetKeyForValidator(0), h[:], bls.DomainVote)
	if err != nil {
		t.Fatal(err)
	}

	voteInvalidSignature := primitives.AggregatedVote{
		Data:          proposalVote,
		Signature:     [48]byte{},
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8),
	}
	voteInvalidSignature.Participation[0] = 1 << 0

	voteNoSignatureFromProposer := primitives.AggregatedVote{
		Data:          proposalVote,
		Signature:     signatureValidator1.Serialize(),
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8),
	}
	voteNoSignatureFromProposer.Participation[0] = 1 << 1

	voteWrongParticipationLength := primitives.AggregatedVote{
		Data:          proposalVote,
		Signature:     signatureValidator0.Serialize(),
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8-1),
	}
	voteWrongParticipationLength.Participation[0] = 1 << 0

	validProposal := primitives.AggregatedVote{
		Data:          proposalVote,
		Signature:     signatureValidator0.Serialize(),
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8),
	}
	validProposal.Participation[0] = 1 << 0

	type testProposal struct {
		proposal   primitives.AggregatedVote
		shouldPass bool
		reason     string
	}

	// invalid signature - fail
	// no signature from proposer on proposal - fail
	// incorrect participation length - fail
	// correct signature - pass
	testProposals := []testProposal{
		{
			proposal:   voteInvalidSignature,
			shouldPass: false,
			reason:     "proposal with invalid signature should not validate",
		},
		{
			proposal:   voteNoSignatureFromProposer,
			shouldPass: false,
			reason:     "proposal with no signature from the proposer should not validate",
		},
		{
			proposal:   voteWrongParticipationLength,
			shouldPass: false,
			reason:     "proposal with wrong participation length should not validate",
		},
		{
			proposal:   validProposal,
			shouldPass: true,
			reason:     "proposal with correct information should validate",
		},
	}

	for _, b := range testProposals {
		blockTest := &primitives.Block{
			BlockHeader: primitives.BlockHeader{
				SlotNumber:     slotToPropose,
				ParentRoot:     chainhash.Hash{},
				StateRoot:      chainhash.Hash{},
				RandaoReveal:   [48]byte{},
				Signature:      [48]byte{},
				ValidatorIndex: proposerIndex,
			},
			BlockBody: primitives.BlockBody{
				Attestations:      nil,
				ProposerSlashings: nil,
				CasperSlashings:   nil,
				Deposits:          nil,
				Exits:             nil,
				Votes:             []primitives.AggregatedVote{b.proposal},
			},
		}

		err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
		if err != nil {
			t.Fatal(err)
		}

		stateCopy := state.Copy()

		err = stateCopy.ProcessBlock(blockTest, c, FakeBlockView{}, true)

		if (err == nil) != b.shouldPass {
			if b.shouldPass {
				t.Fatalf("expected block to validate, but got error: %s for case: %s", err, b.reason)
			} else {
				t.Fatalf("expected block to fail with reason: %s", b.reason)
			}
		}
	}

	// now, actually process the valid proposal
	blockTest := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
			Votes:             []primitives.AggregatedVote{validProposal},
		},
	}

	err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
	if err != nil {
		t.Fatal(err)
	}

	balanceBefore := state.GetEffectiveBalance(0, c)

	err = state.ProcessBlock(blockTest, c, FakeBlockView{}, true)
	if err != nil {
		t.Fatal(err)
	}

	balanceAfter := state.GetEffectiveBalance(0, c)

	slotToPropose++

	proposerIndex, err = state.GetBeaconProposerIndex(slotToPropose-1, c)
	if err != nil {
		t.Fatal(err)
	}

	// cost should be deducted
	// proposal should be added

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	if len(state.Proposals) != 1 {
		t.Fatal("expected proposal to be added after valid proposal")
	}

	if balanceBefore-balanceAfter != c.ProposalCost {
		t.Fatal("expected proposal cost to be deducted from proposer's balance")
	}

	// no signature from proposer on subsequent vote - pass
	// cancel proposal - pass
	// cancel proposal with non-empty shards array - fail
	// cancel proposal with invalid hash - fail
	noSignatureFromProposerOnSubsequentVote := primitives.AggregatedVote{
		Data:          proposalVote,
		Signature:     signatureValidator1.Serialize(),
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8),
	}
	noSignatureFromProposerOnSubsequentVote.Participation[0] = 1 << 1

	cancelData := primitives.VoteData{
		Type:       primitives.Cancel,
		ActionHash: h,
		Shards:     []uint32{},
		Proposer:   1,
	}

	cancelDataHash, _ := ssz.HashTreeRoot(cancelData)

	cancelSignatureValidator1, err := bls.Sign(keystore.GetKeyForValidator(1), cancelDataHash[:], bls.DomainVote)
	if err != nil {
		t.Fatal(err)
	}

	cancelProposal := primitives.AggregatedVote{
		Data:          cancelData,
		Signature:     cancelSignatureValidator1.Serialize(),
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8),
	}
	cancelProposal.Participation[0] = 1 << 1

	cancelDataNonemptyShards := primitives.VoteData{
		Type:       primitives.Cancel,
		ActionHash: h,
		Shards:     []uint32{1},
		Proposer:   1,
	}

	cancelDataNonemptyShardsHash, _ := ssz.HashTreeRoot(cancelData)

	cancelSignatureNonEmptyShards, err := bls.Sign(keystore.GetKeyForValidator(1), cancelDataNonemptyShardsHash[:], bls.DomainVote)
	if err != nil {
		t.Fatal(err)
	}

	cancelProposalNonEmptyShards := primitives.AggregatedVote{
		Data:          cancelDataNonemptyShards,
		Signature:     cancelSignatureNonEmptyShards.Serialize(),
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8),
	}
	cancelProposalNonEmptyShards.Participation[0] = 1 << 1

	cancelDataInvalidHash := primitives.VoteData{
		Type:       primitives.Cancel,
		ActionHash: chainhash.Hash{},
		Shards:     []uint32{},
		Proposer:   1,
	}

	cancelDataInvalidHashHash, _ := ssz.HashTreeRoot(cancelDataInvalidHash)

	cancelDataInvalidHashSignature, err := bls.Sign(keystore.GetKeyForValidator(1), cancelDataInvalidHashHash[:], bls.DomainVote)
	if err != nil {
		t.Fatal(err)
	}

	cancelProposalInvalidHash := primitives.AggregatedVote{
		Data:          cancelDataInvalidHash,
		Signature:     cancelDataInvalidHashSignature.Serialize(),
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8),
	}
	cancelProposalInvalidHash.Participation[0] = 1 << 1

	testProposals = []testProposal{
		{
			proposal:   noSignatureFromProposerOnSubsequentVote,
			shouldPass: true,
			reason:     "proposal with no signature from the vote proposer should pass if there is already an active vote",
		},
		{
			proposal:   cancelProposal,
			shouldPass: true,
			reason:     "normal cancel proposal should validate",
		},
		{
			proposal:   cancelProposalNonEmptyShards,
			shouldPass: false,
			reason:     "cancel proposal with a non-empty shards array should not validate",
		},
		{
			proposal:   cancelProposalInvalidHash,
			shouldPass: false,
			reason:     "cancel proposal with an invalid hash should not validate",
		},
	}

	for _, b := range testProposals {
		blockTest := &primitives.Block{
			BlockHeader: primitives.BlockHeader{
				SlotNumber:     slotToPropose,
				ParentRoot:     chainhash.Hash{},
				StateRoot:      chainhash.Hash{},
				RandaoReveal:   [48]byte{},
				Signature:      [48]byte{},
				ValidatorIndex: proposerIndex,
			},
			BlockBody: primitives.BlockBody{
				Attestations:      nil,
				ProposerSlashings: nil,
				CasperSlashings:   nil,
				Deposits:          nil,
				Exits:             nil,
				Votes:             []primitives.AggregatedVote{b.proposal},
			},
		}

		err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
		if err != nil {
			t.Fatal(err)
		}

		stateCopy := state.Copy()

		err = stateCopy.ProcessBlock(blockTest, c, FakeBlockView{}, true)

		if (err == nil) != b.shouldPass {
			if b.shouldPass {
				t.Fatalf("expected block to validate, but got error: %s for case: %s", err, b.reason)
			} else {
				t.Fatalf("expected block to fail with reason: %s", b.reason)
			}
		}
	}
}

func TestProposerSlashingPenalty(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	// sign two conflicting blocks at the same height
	slotToPropose := state.Slot + 1
	proposerIndex, err := state.GetBeaconProposerIndex(slotToPropose, c)
	if err != nil {
		t.Fatal(err)
	}

	block1 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose + 1,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	key := keystore.GetKeyForValidator(proposerIndex)

	var slotBytes [8]byte
	binary.BigEndian.PutUint64(slotBytes[:], block1.BlockHeader.SlotNumber)
	slotBytesHash := chainhash.HashH(slotBytes[:])

	slotBytesSignature, err := bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		t.Fatal(err)
	}

	block1.BlockHeader.RandaoReveal = slotBytesSignature.Serialize()
	block1.BlockHeader.Signature = bls.EmptySignature.Serialize()

	blockWithoutSignatureRoot, err := ssz.HashTreeRoot(block1)
	if err != nil {
		t.Fatal(err)
	}

	proposal1 := primitives.ProposalSignedData{
		Slot:      block1.BlockHeader.SlotNumber,
		Shard:     c.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot1, err := ssz.HashTreeRoot(proposal1)
	if err != nil {
		t.Fatal(err)
	}

	blockSignature1, err := bls.Sign(key, proposalRoot1[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	block2 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose + 1,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      []primitives.Attestation{{}},
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	binary.BigEndian.PutUint64(slotBytes[:], block2.BlockHeader.SlotNumber)
	slotBytesHash = chainhash.HashH(slotBytes[:])

	slotBytesSignature, err = bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		t.Fatal(err)
	}

	block2.BlockHeader.RandaoReveal = slotBytesSignature.Serialize()
	block2.BlockHeader.Signature = bls.EmptySignature.Serialize()

	blockWithoutSignatureRoot, err = ssz.HashTreeRoot(block2)
	if err != nil {
		t.Fatal(err)
	}

	proposal2 := primitives.ProposalSignedData{
		Slot:      block2.BlockHeader.SlotNumber,
		Shard:     c.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot2, err := ssz.HashTreeRoot(proposal2)
	if err != nil {
		t.Fatal(err)
	}

	blockSignature2, err := bls.Sign(key, proposalRoot2[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	slashing := primitives.ProposerSlashing{
		ProposerIndex:      proposerIndex,
		ProposalData1:      proposal1,
		ProposalSignature1: blockSignature1.Serialize(),
		ProposalData2:      proposal2,
		ProposalSignature2: blockSignature2.Serialize(),
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	slashedProposerIndex := proposerIndex

	proposerIndex, err = state.GetBeaconProposerIndex(slotToPropose-1, c)
	if err != nil {
		t.Fatal(err)
	}

	blockTest := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations: nil,
			ProposerSlashings: []primitives.ProposerSlashing{
				slashing,
			},
			CasperSlashings: nil,
			Deposits:        nil,
			Exits:           nil,
		},
	}

	err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
	if err != nil {
		t.Fatal(err)
	}

	stateCopy := state.Copy()

	err = stateCopy.ProcessBlock(blockTest, c, FakeBlockView{}, true)
	if err != nil {
		t.Fatal(err)
	}

	slashedLessBalance := state.ValidatorBalances[slashedProposerIndex] - stateCopy.ValidatorBalances[slashedProposerIndex]

	slashedBalanceBefore := state.GetEffectiveBalance(slashedProposerIndex, c)

	expectedSlash := slashedBalanceBefore / c.WhistleblowerRewardQuotient

	if slashedLessBalance != expectedSlash {
		t.Fatalf("wrong slashed amount. (slashed is %d, should be %d)", slashedLessBalance, expectedSlash)
	}

	whistleblowerMoreBalance := stateCopy.ValidatorBalances[proposerIndex] - state.ValidatorBalances[proposerIndex]

	if whistleblowerMoreBalance != expectedSlash {
		t.Fatalf("wrong whistleblower amount. (whistleblower got %d, should be %d)", whistleblowerMoreBalance, expectedSlash)
	}
}

func TestVoteParticipationSliceGrowth(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	slotToPropose := state.Slot + 1
	proposerIndex, err := state.GetBeaconProposerIndex(slotToPropose-1, c)
	if err != nil {
		t.Fatal(err)
	}

	block1 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	key := keystore.GetKeyForValidator(proposerIndex)

	err = SignBlock(block1, key, c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	proposalVote := primitives.VoteData{
		Type:       primitives.Propose,
		Shards:     []uint32{1, 2},
		ActionHash: chainhash.Hash{},
		Proposer:   0,
	}

	h, _ := ssz.HashTreeRoot(proposalVote)

	signatureValidator0, err := bls.Sign(keystore.GetKeyForValidator(0), h[:], bls.DomainVote)
	if err != nil {
		t.Fatal(err)
	}

	validProposal := primitives.AggregatedVote{
		Data:          proposalVote,
		Signature:     signatureValidator0.Serialize(),
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8),
	}
	validProposal.Participation[0] = 1 << 0

	// now, actually process the valid proposal
	blockTest := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
			Votes:             []primitives.AggregatedVote{validProposal},
		},
	}

	err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessBlock(blockTest, c, FakeBlockView{}, true)
	if err != nil {
		t.Fatal(err)
	}

	slotToPropose++

	proposerIndex, err = state.GetBeaconProposerIndex(slotToPropose-1, c)
	if err != nil {
		t.Fatal(err)
	}

	startValidatorIndex := len(state.ValidatorRegistry)

	for i := 0; i < 20; i++ {
		pubkey := keystore.GetPublicKeyForValidator(uint32(startValidatorIndex + i))
		state.ValidatorRegistry = append(state.ValidatorRegistry, primitives.Validator{
			Pubkey:                  pubkey.Serialize(),
			WithdrawalCredentials:   chainhash.Hash{},
			Status:                  primitives.Active,
			LatestStatusChangeSlot:  0,
			ExitCount:               0,
			LastPoCChangeSlot:       0,
			SecondLastPoCChangeSlot: 0,
		})
	}

	// now that we entered a few validators, the participation array should get bigger

	lastValidatorIndex := uint32(startValidatorIndex + 19)

	signatureLastValidator, err := bls.Sign(keystore.GetKeyForValidator(lastValidatorIndex), h[:], bls.DomainVote)
	if err != nil {
		t.Fatal(err)
	}

	lastValidatorProposal := primitives.AggregatedVote{
		Data:          proposalVote,
		Signature:     signatureLastValidator.Serialize(),
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8),
	}
	lastValidatorProposal.Participation[lastValidatorIndex/8] = 1 << uint(lastValidatorIndex%8)

	blockTest = &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose,
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
			Votes:             []primitives.AggregatedVote{lastValidatorProposal},
		},
	}

	err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)

	err = state.ProcessBlock(blockTest, c, FakeBlockView{}, true)
	if err != nil {
		t.Fatal(err)
	}

	if state.Proposals[0].Participation[lastValidatorIndex/8]&(1<<uint(lastValidatorIndex%8)) == 0 {
		t.Fatal("expected vote to be processed")
	}
}

func TestVoteValidatorLeave(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	slotToPropose := state.Slot + 1
	proposerIndex, err := state.GetBeaconProposerIndex(slotToPropose-1, c)
	if err != nil {
		t.Fatal(err)
	}

	block1 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotToPropose,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: [48]byte{},
			Signature:    [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	key := keystore.GetKeyForValidator(proposerIndex)

	err = SignBlock(block1, key, c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	proposalVote := primitives.VoteData{
		Type:       primitives.Propose,
		Shards:     []uint32{1, 2},
		ActionHash: chainhash.Hash{},
		Proposer:   0,
	}

	h, _ := ssz.HashTreeRoot(proposalVote)

	signatureValidator0, err := bls.Sign(keystore.GetKeyForValidator(0), h[:], bls.DomainVote)
	if err != nil {
		t.Fatal(err)
	}

	validProposal := primitives.AggregatedVote{
		Data:          proposalVote,
		Signature:     signatureValidator0.Serialize(),
		Participation: make([]uint8, (len(state.ValidatorRegistry)+7)/8),
	}
	validProposal.Participation[0] = 1 << 0

	// now, actually process the valid proposal
	blockTest := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     slotToPropose,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: proposerIndex,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
			Votes:             []primitives.AggregatedVote{validProposal},
		},
	}

	err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessBlock(blockTest, c, FakeBlockView{}, true)
	if err != nil {
		t.Fatal(err)
	}

	if state.Proposals[0].Participation[0]&1 == 0 {
		t.Fatal("expected vote to be processed")
	}

	err = state.ExitValidator(0, primitives.ExitedWithoutPenalty, c)
	if err != nil {
		t.Fatal(err)
	}

	if state.Proposals[0].Participation[0]&1 != 0 {
		t.Fatal("expected vote to be reset on validator exit")
	}
}
