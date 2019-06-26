package primitives_test

import (
	"testing"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"

	"github.com/go-test/deep"
)

func TestVoteData_Copy(t *testing.T) {
	baseVoteData := &primitives.VoteData{
		Type:       0,
		Shards:     []byte{0},
		ActionHash: chainhash.Hash{},
		Proposer:   0,
	}

	copyVoteData := baseVoteData.Copy()

	copyVoteData.Type = 1
	if baseVoteData.Type == 1 {
		t.Fatal("mutating type mutates base")
	}

	copyVoteData.Shards[0] = 1
	if baseVoteData.Shards[0] == 1 {
		t.Fatal("mutating shards mutates base")
	}

	copyVoteData.ActionHash[0] = 1
	if baseVoteData.ActionHash[0] == 1 {
		t.Fatal("mutating actionHash mutates base")
	}

	copyVoteData.Proposer = 1
	if baseVoteData.Proposer == 1 {
		t.Fatal("mutating proposer mutates base")
	}
}

func TestVoteData_ToFromProto(t *testing.T) {
	baseVoteData := &primitives.VoteData{
		Type:       1,
		Shards:     []byte{1},
		ActionHash: chainhash.Hash{1},
		Proposer:   1,
	}

	voteDataProto := baseVoteData.ToProto()
	fromProto, err := primitives.VoteDataFromProto(voteDataProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseVoteData); diff != nil {
		t.Fatal(diff)
	}
}

func TestAggregatedVote_Copy(t *testing.T) {
	baseAggregatedVote := &primitives.AggregatedVote{
		Data: primitives.VoteData{
			Type:       0,
			Shards:     nil,
			ActionHash: chainhash.Hash{},
			Proposer:   0,
		},
		Signature:     [48]byte{},
		Participation: []uint8{0},
	}

	copyAggregatedVote := baseAggregatedVote.Copy()

	copyAggregatedVote.Data.Type = 1
	if baseAggregatedVote.Data.Type == 1 {
		t.Fatal("mutating data mutates base")
	}

	copyAggregatedVote.Signature[0] = 1
	if baseAggregatedVote.Signature[0] == 1 {
		t.Fatal("mutating signature mutates base")
	}

	copyAggregatedVote.Participation[0] = 1
	if baseAggregatedVote.Participation[0] == 1 {
		t.Fatal("mutating participation mutates base")
	}
}

func TestAggregatedVote_ToFromProto(t *testing.T) {
	baseAggregatedVote := &primitives.AggregatedVote{
		Data: primitives.VoteData{
			Type:       0,
			Shards:     nil,
			ActionHash: chainhash.Hash{},
			Proposer:   0,
		},
		Signature:     [48]byte{},
		Participation: []uint8{0},
	}

	aggregatedVoteProto := baseAggregatedVote.ToProto()
	fromProto, err := primitives.AggregatedVoteFromProto(aggregatedVoteProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseAggregatedVote); diff != nil {
		t.Fatal(diff)
	}
}

func TestActiveProposal_Copy(t *testing.T) {
	baseActiveProposal := &primitives.ActiveProposal{
		Data: primitives.VoteData{
			Type:       0,
			Shards:     nil,
			ActionHash: chainhash.Hash{},
			Proposer:   0,
		},
		Participation: []uint8{0},
		StartEpoch:    0,
		Queued:        false,
	}

	copyActiveProposal := baseActiveProposal.Copy()

	copyActiveProposal.Data.Type = 1
	if baseActiveProposal.Data.Type == 1 {
		t.Fatal("mutating data mutates base")
	}

	copyActiveProposal.Participation[0] = 1
	if baseActiveProposal.Participation[0] == 1 {
		t.Fatal("mutating participation mutates base")
	}

	copyActiveProposal.StartEpoch = 1
	if baseActiveProposal.StartEpoch == 1 {
		t.Fatal("mutating startepoch mutates base")
	}

	copyActiveProposal.Queued = true
	if baseActiveProposal.Queued == true {
		t.Fatal("mutating queued mutates base")
	}
}

func TestActiveProposal_ToFromProto(t *testing.T) {
	baseActiveProposal := &primitives.ActiveProposal{
		Data: primitives.VoteData{
			Type:       1,
			Shards:     []byte{1},
			ActionHash: chainhash.Hash{1},
			Proposer:   1,
		},
		Participation: []uint8{1},
		StartEpoch:    1,
		Queued:        true,
	}

	activeProposalProto := baseActiveProposal.ToProto()
	fromProto, err := primitives.ActiveProposalFromProto(activeProposalProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseActiveProposal); diff != nil {
		t.Fatal(diff)
	}
}
