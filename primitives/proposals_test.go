package primitives_test

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

func TestProposalSignedData_Copy(t *testing.T) {
	proposalSignedData := &primitives.ProposalSignedData{
		Slot:      0,
		Shard:     0,
		BlockHash: chainhash.Hash{},
	}

	copyProposalSignedData := proposalSignedData.Copy()

	copyProposalSignedData.Slot = 1
	if proposalSignedData.Slot == 1 {
		t.Fatal("mutating slot mutates base")
	}

	copyProposalSignedData.Shard = 1
	if proposalSignedData.Shard == 1 {
		t.Fatal("mutating shard mutates base")
	}

	copyProposalSignedData.BlockHash[0] = 1
	if proposalSignedData.BlockHash[0] == 1 {
		t.Fatal("mutating blockHash mutates base")
	}
}

func TestProposalSignedData_ToFromProto(t *testing.T) {
	proposalSignedData := &primitives.ProposalSignedData{
		Slot:      1,
		Shard:     1,
		BlockHash: chainhash.Hash{1},
	}

	proposalDataProto := proposalSignedData.ToProto()
	fromProto, err := primitives.ProposalSignedDataFromProto(proposalDataProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, proposalSignedData); diff != nil {
		t.Fatal(diff)
	}
}
