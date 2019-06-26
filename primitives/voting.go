package primitives

import (
	"errors"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
)

const (
	// Propose indicates the vote is proposing a change
	Propose = iota

	// Cancel indicates the vote is cancelling a queued proposal.
	Cancel
)

// VoteData represents what the vote means. It includes information about what shards will
// be affected, the hash of the code (for proposals) or proposal hash (for cancellations),
// and the proposer validator ID.
type VoteData struct {
	Type       uint8
	Shards     []byte
	ActionHash chainhash.Hash
	Proposer   uint32
}

// ToProto converts the VoteData into a protobuf representation.
func (vd *VoteData) ToProto() *pb.VoteData {
	return &pb.VoteData{
		Type:       uint32(vd.Type),
		Shards:     vd.Shards,
		ActionHash: vd.ActionHash[:],
		Proposer:   vd.Proposer,
	}
}

// VoteDataFromProto unwraps a protobuf representation of vote data.
func VoteDataFromProto(vd *pb.VoteData) (*VoteData, error) {
	if vd.Type > 255 {
		return nil, errors.New("vote data type should be less than 255")
	}

	var actionHash chainhash.Hash

	t := uint8(vd.Type)

	err := actionHash.SetBytes(vd.ActionHash)
	if err != nil {
		return nil, err
	}

	return &VoteData{
		Type:       t,
		Shards:     vd.Shards,
		ActionHash: actionHash,
		Proposer:   vd.Proposer,
	}, nil
}

// Copy returns a copy of the vote data.
func (vd *VoteData) Copy() VoteData {
	v := *vd

	v.Shards = append([]uint8{}, vd.Shards...)

	return v
}
