package primitives_test

import (
	"testing"

	"github.com/phoreproject/synapse/chainhash"

	"github.com/go-test/deep"
	"github.com/phoreproject/synapse/primitives"
)

func TestProposerSlashing_Copy(t *testing.T) {
	baseProposerSlashing := &primitives.ProposerSlashing{
		ProposerIndex:      0,
		ProposalData2:      primitives.ProposalSignedData{},
		ProposalData1:      primitives.ProposalSignedData{},
		ProposalSignature1: [48]byte{},
		ProposalSignature2: [48]byte{},
	}

	copyProposerSlashing := baseProposerSlashing.Copy()

	copyProposerSlashing.ProposerIndex = 1
	if baseProposerSlashing.ProposerIndex == 1 {
		t.Fatal("mutating attestations mutates base")
	}

	copyProposerSlashing.ProposalData1.Slot = 1
	if baseProposerSlashing.ProposalData1.Slot == 1 {
		t.Fatal("mutating proposalData1 mutates base")
	}

	copyProposerSlashing.ProposalData2.Slot = 1
	if baseProposerSlashing.ProposalData2.Slot == 1 {
		t.Fatal("mutating proposalData2 mutates base")
	}

	copyProposerSlashing.ProposalSignature1[0] = 1
	if baseProposerSlashing.ProposalSignature1[0] == 1 {
		t.Fatal("mutating proposalSignature1 mutates base")
	}

	copyProposerSlashing.ProposalSignature2[0] = 1
	if baseProposerSlashing.ProposalSignature2[0] == 1 {
		t.Fatal("mutating proposalSignature2 mutates base")
	}
}

func TestProposerSlashing_ToFromProto(t *testing.T) {
	baseProposalSlashing := &primitives.ProposerSlashing{
		ProposerIndex:      1,
		ProposalData2:      primitives.ProposalSignedData{Slot: 1},
		ProposalData1:      primitives.ProposalSignedData{Slot: 1},
		ProposalSignature1: [48]byte{1},
		ProposalSignature2: [48]byte{1},
	}

	proposalSlashingProto := baseProposalSlashing.ToProto()
	fromProto, err := primitives.ProposerSlashingFromProto(proposalSlashingProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseProposalSlashing); diff != nil {
		t.Fatal(diff)
	}
}

func TestSlashableVoteData_Copy(t *testing.T) {
	baseSlashableVoteData := &primitives.SlashableVoteData{
		AggregateSignaturePoC0Indices: []uint32{0, 0, 0},
		AggregateSignaturePoC1Indices: []uint32{0, 0, 0},
		Data: primitives.AttestationData{
			Slot:                0,
			Shard:               0,
			BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
			SourceEpoch:         0,
			SourceHash:          chainhash.HashH([]byte("epoch")),
			ShardBlockHash:      chainhash.HashH([]byte("shard")),
			LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
			TargetEpoch:         1,
			TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
		},
		AggregateSignature: [48]byte{},
	}

	copySlashableVoteData := baseSlashableVoteData.Copy()

	copySlashableVoteData.AggregateSignaturePoC0Indices[0] = 1
	if baseSlashableVoteData.AggregateSignaturePoC0Indices[0] == 1 {
		t.Fatal("mutating AggregateSignaturePoC0Indices mutates base")
	}

	copySlashableVoteData.AggregateSignaturePoC1Indices[0] = 1
	if baseSlashableVoteData.AggregateSignaturePoC1Indices[0] == 1 {
		t.Fatal("mutating AggregateSignaturePoC1Indices mutates base")
	}

	copySlashableVoteData.Data.Slot = 1
	if baseSlashableVoteData.Data.Slot == 1 {
		t.Fatal("mutating attestationData mutates base")
	}

	copySlashableVoteData.AggregateSignature[0] = 1
	if baseSlashableVoteData.AggregateSignature[0] == 1 {
		t.Fatal("mutating aggregateSignature mutates base")
	}
}

func TestSlashableVoteData_ToFromProto(t *testing.T) {
	baseSlashableVoteData := &primitives.SlashableVoteData{
		AggregateSignaturePoC0Indices: []uint32{0, 0, 0},
		AggregateSignaturePoC1Indices: []uint32{0, 0, 0},
		Data: primitives.AttestationData{
			Slot:                0,
			Shard:               0,
			BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
			SourceEpoch:         0,
			SourceHash:          chainhash.HashH([]byte("epoch")),
			ShardBlockHash:      chainhash.HashH([]byte("shard")),
			LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
			TargetEpoch:         1,
			TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
		},
		AggregateSignature: [48]byte{},
	}

	slashableVoteDataProto := baseSlashableVoteData.ToProto()
	fromProto, err := primitives.SlashableVoteDataFromProto(slashableVoteDataProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseSlashableVoteData); diff != nil {
		t.Fatal(diff)
	}
}

func TestCasperSlashing_Copy(t *testing.T) {
	baseCasperSlashing := &primitives.CasperSlashing{
		Votes1: primitives.SlashableVoteData{
			AggregateSignaturePoC0Indices: []uint32{0, 0, 0},
			AggregateSignaturePoC1Indices: []uint32{0, 0, 0},
			Data: primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         1,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			AggregateSignature: [48]byte{},
		},
		Votes2: primitives.SlashableVoteData{
			AggregateSignaturePoC0Indices: []uint32{0, 0, 0},
			AggregateSignaturePoC1Indices: []uint32{0, 0, 0},
			Data: primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         1,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			AggregateSignature: [48]byte{},
		},
	}

	copyCasperSlashing := baseCasperSlashing.Copy()

	copyCasperSlashing.Votes1.AggregateSignature[0] = 1
	if baseCasperSlashing.Votes1.AggregateSignature[0] == 1 {
		t.Fatal("mutating Votes1 mutates base")
	}

	copyCasperSlashing.Votes2.AggregateSignature[0] = 1
	if baseCasperSlashing.Votes2.AggregateSignature[0] == 1 {
		t.Fatal("mutating Votes2 mutates base")
	}
}

func TestCasperSlashing_ToFromProto(t *testing.T) {
	baseCasperSlashing := &primitives.CasperSlashing{
		Votes1: primitives.SlashableVoteData{
			AggregateSignaturePoC0Indices: []uint32{0, 0, 0},
			AggregateSignaturePoC1Indices: []uint32{0, 0, 0},
			Data: primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         1,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			AggregateSignature: [48]byte{},
		},
		Votes2: primitives.SlashableVoteData{
			AggregateSignaturePoC0Indices: []uint32{0, 0, 0},
			AggregateSignaturePoC1Indices: []uint32{0, 0, 0},
			Data: primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         1,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			AggregateSignature: [48]byte{},
		},
	}

	casperSlashingProto := baseCasperSlashing.ToProto()
	fromProto, err := primitives.CasperSlashingFromProto(casperSlashingProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseCasperSlashing); diff != nil {
		t.Fatal(diff)
	}
}
