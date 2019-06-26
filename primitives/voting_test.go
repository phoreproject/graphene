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
