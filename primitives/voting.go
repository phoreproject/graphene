package primitives

import (
	"errors"

	"github.com/phoreproject/graphene/chainhash"
	"github.com/phoreproject/graphene/pb"
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
	Shards     []uint32
	ActionHash chainhash.Hash
	Proposer   uint32
}

// ToProto converts the VoteData into a protobuf representation.
func (vd *VoteData) ToProto() *pb.VoteData {
	return &pb.VoteData{
		Type:       uint32(vd.Type),
		Shards:     vd.Shards[:],
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
	v.Shards = append([]uint32{}, vd.Shards...)
	return v
}

// AggregatedVote represents voting data and an arbitrary number of votes from the active
// validator set.
type AggregatedVote struct {
	Data          VoteData
	Signature     [48]byte
	Participation []byte
}

// ToProto converts the AggregatedVoted into a protobuf representation.
func (av *AggregatedVote) ToProto() *pb.AggregatedVote {
	return &pb.AggregatedVote{
		Data:          av.Data.ToProto(),
		Signature:     av.Signature[:],
		Participation: av.Participation[:],
	}
}

// AggregatedVoteFromProto unwraps a protobuf representation of vote data.
func AggregatedVoteFromProto(vd *pb.AggregatedVote) (*AggregatedVote, error) {
	data, err := VoteDataFromProto(vd.Data)
	if err != nil {
		return nil, err
	}

	if len(vd.Signature) != 48 {
		return nil, errors.New("signature must be 48 bytes long")
	}

	var sigBytes [48]byte
	copy(sigBytes[:], vd.Signature)

	return &AggregatedVote{
		Data:          *data,
		Signature:     sigBytes,
		Participation: vd.Participation,
	}, nil
}

// Copy returns a copy of the vote data.
func (av *AggregatedVote) Copy() AggregatedVote {
	v := *av
	v.Data = av.Data.Copy()
	v.Participation = append([]uint8{}, av.Participation...)
	return v
}

// ActiveProposal represents a proposal to implement new code.
type ActiveProposal struct {
	Data          VoteData
	Participation []uint8
	StartEpoch    uint64
	Queued        bool
}

// ToProto converts the ActiveProposal into a protobuf representation.
func (ap *ActiveProposal) ToProto() *pb.ActiveProposal {
	return &pb.ActiveProposal{
		Data:          ap.Data.ToProto(),
		StartEpoch:    ap.StartEpoch,
		Participation: ap.Participation[:],
		Queued:        ap.Queued,
	}
}

// Copy returns a copy of the active proposal.
func (ap *ActiveProposal) Copy() ActiveProposal {
	newAp := *ap
	newAp.Data = ap.Data.Copy()
	newAp.Participation = append([]uint8{}, ap.Participation...)
	return newAp
}

// ActiveProposalFromProto unwraps a protobuf representation of active proposal.
func ActiveProposalFromProto(ap *pb.ActiveProposal) (*ActiveProposal, error) {
	data, err := VoteDataFromProto(ap.Data)
	if err != nil {
		return nil, err
	}

	return &ActiveProposal{
		Data:          *data,
		Participation: ap.Participation,
		StartEpoch:    ap.StartEpoch,
		Queued:        ap.Queued,
	}, nil
}
