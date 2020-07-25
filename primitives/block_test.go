package primitives_test

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

func TestBlock_Copy(t *testing.T) {
	baseBlock := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     0,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
			ValidatorIndex: 0,
		},
		BlockBody: primitives.BlockBody{
			Attestations:      []primitives.Attestation{},
			ProposerSlashings: []primitives.ProposerSlashing{},
			CasperSlashings:   []primitives.CasperSlashing{},
			Deposits:          []primitives.Deposit{},
			Exits:             []primitives.Exit{},
			Votes:             []primitives.AggregatedVote{},
		},
	}

	copyBlock := baseBlock.Copy()

	copyBlock.BlockHeader.SlotNumber = 1
	if baseBlock.BlockHeader.SlotNumber == 1 {
		t.Fatal("mutating block header mutates base")
	}

	copyBlock.BlockBody.Attestations = nil
	if baseBlock.BlockBody.Attestations == nil {
		t.Fatal("mutating block body mutates base")
	}
}

func TestBlock_ToFromProto(t *testing.T) {
	baseBlock := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     0,
			ValidatorIndex: 0,
			ParentRoot:     chainhash.Hash{},
			StateRoot:      chainhash.Hash{},
			RandaoReveal:   [48]byte{},
			Signature:      [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations:      []primitives.Attestation{},
			ProposerSlashings: []primitives.ProposerSlashing{},
			CasperSlashings:   []primitives.CasperSlashing{},
			Deposits:          []primitives.Deposit{},
			Exits:             []primitives.Exit{},
			Votes:             []primitives.AggregatedVote{},
		},
	}

	blockProto := baseBlock.ToProto()
	fromProto, err := primitives.BlockFromProto(blockProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseBlock); diff != nil {
		t.Fatal(diff)
	}
}

func TestBlockHeader_Copy(t *testing.T) {
	baseBlockHeader := &primitives.BlockHeader{
		SlotNumber:     0,
		ValidatorIndex: 0,
		ParentRoot:     chainhash.Hash{},
		StateRoot:      chainhash.Hash{},
		RandaoReveal:   [48]byte{},
		Signature:      [48]byte{},
		CurrentValidatorHash: chainhash.Hash{},
	}

	copyBlockHeader := baseBlockHeader.Copy()

	copyBlockHeader.ValidatorIndex = 1
	if baseBlockHeader.ValidatorIndex == 1 {
		t.Fatal("mutating validator index mutates base")
	}

	copyBlockHeader.SlotNumber = 1
	if baseBlockHeader.SlotNumber == 1 {
		t.Fatal("mutating slotnumber mutates base")
	}

	copyBlockHeader.ParentRoot[0] = 1
	if baseBlockHeader.ParentRoot[0] == 1 {
		t.Fatal("mutating parentroot mutates base")
	}

	copyBlockHeader.StateRoot[0] = 1
	if baseBlockHeader.StateRoot[0] == 1 {
		t.Fatal("mutating stateroot mutates base")
	}

	copyBlockHeader.RandaoReveal[0] = 1
	if baseBlockHeader.RandaoReveal[0] == 1 {
		t.Fatal("mutating randaoreveal mutates base")
	}

	copyBlockHeader.Signature[0] = 1
	if baseBlockHeader.Signature[0] == 1 {
		t.Fatal("mutating signature mutates base")
	}

	copyBlockHeader.CurrentValidatorHash[0] = 1
	if baseBlockHeader.CurrentValidatorHash[0] == 1 {
		t.Fatal("mutating CurrentValidatorHash mutates base")
	}
}

func TestBlockHeader_ToFromProto(t *testing.T) {
	baseBlock := &primitives.BlockHeader{
		SlotNumber:     1,
		ParentRoot:     chainhash.Hash{2},
		StateRoot:      chainhash.Hash{3},
		RandaoReveal:   [48]byte{4},
		Signature:      [48]byte{5},
		ValidatorIndex: 6,
		CurrentValidatorHash: chainhash.Hash{7},
	}

	blockHeaderProto := baseBlock.ToProto()
	fromProto, err := primitives.BlockHeaderFromProto(blockHeaderProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseBlock); diff != nil {
		t.Fatal(diff)
	}
}

func TestBlockBody_Copy(t *testing.T) {
	baseBlockBody := &primitives.BlockBody{
		Attestations:      []primitives.Attestation{{}},
		ProposerSlashings: []primitives.ProposerSlashing{{}},
		CasperSlashings:   []primitives.CasperSlashing{{}},
		Deposits:          []primitives.Deposit{{}},
		Exits:             []primitives.Exit{{}},
		Votes:             []primitives.AggregatedVote{{}},
	}

	copyBlockBody := baseBlockBody.Copy()

	copyBlockBody.Attestations = nil
	if baseBlockBody.Attestations == nil {
		t.Fatal("mutating attestations mutates base")
	}

	copyBlockBody.ProposerSlashings = nil
	if baseBlockBody.ProposerSlashings == nil {
		t.Fatal("mutating proposerSlashings mutates base")
	}

	copyBlockBody.CasperSlashings = nil
	if baseBlockBody.CasperSlashings == nil {
		t.Fatal("mutating casperSlashings mutates base")
	}

	copyBlockBody.Deposits = nil
	if baseBlockBody.Deposits == nil {
		t.Fatal("mutating deposits mutates base")
	}

	copyBlockBody.Exits = nil
	if baseBlockBody.Exits == nil {
		t.Fatal("mutating exits mutates base")
	}

	copyBlockBody.Votes = nil
	if baseBlockBody.Votes == nil {
		t.Fatal("mutating exits mutates base")
	}
}

func TestBlockBody_ToFromProto(t *testing.T) {
	baseBlockBody := &primitives.BlockBody{
		Attestations:      []primitives.Attestation{{ParticipationBitfield: []uint8{1}}},
		ProposerSlashings: []primitives.ProposerSlashing{{ProposerIndex: 1}},
		CasperSlashings: []primitives.CasperSlashing{
			{
				Votes1: primitives.SlashableVoteData{
					AggregateSignature:            [48]byte{1},
					AggregateSignaturePoC0Indices: []uint32{},
					AggregateSignaturePoC1Indices: []uint32{},
				},
				Votes2: primitives.SlashableVoteData{
					AggregateSignature:            [48]byte{1},
					AggregateSignaturePoC0Indices: []uint32{},
					AggregateSignaturePoC1Indices: []uint32{},
				},
			},
		},
		Deposits: []primitives.Deposit{
			{
				Parameters: primitives.DepositParameters{
					PubKey: [96]byte{1},
				},
			},
		},
		Exits: []primitives.Exit{
			{
				Slot: 1,
			},
		},
		Votes: []primitives.AggregatedVote{
			{
				Signature: [48]byte{1},
			},
		},
	}

	blockBodyProto := baseBlockBody.ToProto()
	fromProto, err := primitives.BlockBodyFromProto(blockBodyProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseBlockBody); diff != nil {
		t.Fatal(diff)
	}
}
