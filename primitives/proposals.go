package primitives

import (
	"github.com/phoreproject/graphene/chainhash"
	"github.com/phoreproject/graphene/pb"
)

// ProposalSignedData is a block proposal for a shard or beacon
// chain.
type ProposalSignedData struct {
	Slot      uint64
	Shard     uint64
	BlockHash chainhash.Hash
}

// Copy returns a copy of the proposal signed data.
func (psd *ProposalSignedData) Copy() ProposalSignedData {
	return *psd
}

// ToProto gets the protobuf representation of a proposal signed data object
func (psd *ProposalSignedData) ToProto() *pb.ProposalSignedData {
	return &pb.ProposalSignedData{
		Slot:      psd.Slot,
		Shard:     psd.Shard,
		BlockHash: psd.BlockHash[:],
	}
}

// ProposalSignedDataFromProto gets the proposal for the protobuf representation.
func ProposalSignedDataFromProto(data *pb.ProposalSignedData) (*ProposalSignedData, error) {
	psd := &ProposalSignedData{
		Slot:  data.Slot,
		Shard: data.Shard,
	}
	err := psd.BlockHash.SetBytes(data.BlockHash)
	if err != nil {
		return nil, err
	}
	return psd, nil
}
