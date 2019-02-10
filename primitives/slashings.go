package primitives

import (
	"errors"

	"github.com/phoreproject/synapse/pb"
)

// ProposerSlashing is a slashing request for a proposal violation.
type ProposerSlashing struct {
	ProposerIndex      uint32
	ProposalData1      ProposalSignedData
	ProposalSignature1 [48]byte
	ProposalData2      ProposalSignedData
	ProposalSignature2 [48]byte
}

// Copy returns a copy of the proposer slashing.
func (ps *ProposerSlashing) Copy() ProposerSlashing {
	return *ps
}

// ToProto gets the protobuf representation of the proposer slashing.
func (ps *ProposerSlashing) ToProto() *pb.ProposerSlashing {
	newPS := &pb.ProposerSlashing{
		ProposerIndex: ps.ProposerIndex,
		ProposalData1: ps.ProposalData1.ToProto(),
		ProposalData2: ps.ProposalData2.ToProto(),
	}
	newPS.ProposalSignature1 = ps.ProposalSignature1[:]
	newPS.ProposalSignature2 = ps.ProposalSignature2[:]
	return newPS
}

// ProposerSlashingFromProto gets the proposer slashing from the protobuf representation
func ProposerSlashingFromProto(slashing *pb.ProposerSlashing) (*ProposerSlashing, error) {
	if len(slashing.ProposalSignature1) > 48 {
		return nil, errors.New("proposalSignature1 should be 48 bytes long")
	}
	if len(slashing.ProposalSignature2) > 48 {
		return nil, errors.New("proposalSignature2 should be 48 bytes long")
	}
	pd1, err := ProposalSignedDataFromProto(slashing.ProposalData1)
	if err != nil {
		return nil, err
	}
	pd2, err := ProposalSignedDataFromProto(slashing.ProposalData2)
	if err != nil {
		return nil, err
	}
	ps := &ProposerSlashing{
		ProposalData1: *pd1,
		ProposalData2: *pd2,
		ProposerIndex: slashing.ProposerIndex,
	}
	copy(ps.ProposalSignature1[:], slashing.ProposalSignature1)
	copy(ps.ProposalSignature2[:], slashing.ProposalSignature2)
	return ps, nil
}

// SlashableVoteData is the vote data that should be slashed.
type SlashableVoteData struct {
	AggregateSignaturePoC0Indices []uint32
	AggregateSignaturePoC1Indices []uint32
	Data                          AttestationData
	AggregateSignature            [48]byte
}

// Copy returns a copy of the slashable vote data.
func (svd *SlashableVoteData) Copy() SlashableVoteData {
	return SlashableVoteData{
		AggregateSignaturePoC0Indices: append([]uint32{}, svd.AggregateSignaturePoC0Indices[:]...),
		AggregateSignaturePoC1Indices: append([]uint32{}, svd.AggregateSignaturePoC1Indices[:]...),
		Data:                          svd.Data.Copy(),
		AggregateSignature:            svd.AggregateSignature,
	}
}

// ToProto returns the protobuf representation of the slashable vote data.
func (svd *SlashableVoteData) ToProto() *pb.SlashableVoteData {
	newSvd := &pb.SlashableVoteData{
		AggregateSignaturePoC0Indices: append([]uint32{}, svd.AggregateSignaturePoC0Indices[:]...),
		AggregateSignaturePoC1Indices: append([]uint32{}, svd.AggregateSignaturePoC1Indices[:]...),
		Data:                          svd.Data.ToProto(),
	}
	newSvd.AggregateSignature = append([]byte{}, svd.AggregateSignature[:]...)
	return newSvd
}

// SlashableVoteDataFromProto returns the vote data from the protobuf representation
func SlashableVoteDataFromProto(voteData *pb.SlashableVoteData) (*SlashableVoteData, error) {
	if len(voteData.AggregateSignature) > 48 {
		return nil, errors.New("aggregateSignature should be 48 bytes")
	}
	svd := &SlashableVoteData{}
	svd.AggregateSignaturePoC0Indices = append([]uint32{}, voteData.AggregateSignaturePoC0Indices...)
	svd.AggregateSignaturePoC1Indices = append([]uint32{}, voteData.AggregateSignaturePoC1Indices...)
	copy(svd.AggregateSignature[:], voteData.AggregateSignature)

	data, err := AttestationDataFromProto(voteData.Data)
	if err != nil {
		return nil, err
	}

	svd.Data = *data
	return svd, nil
}

// CasperSlashing is a claim to slash based on two votes.
type CasperSlashing struct {
	Votes1 SlashableVoteData
	Votes2 SlashableVoteData
}

// Copy returns a copy of the casper slashing.
func (cs *CasperSlashing) Copy() CasperSlashing {
	return CasperSlashing{
		cs.Votes1.Copy(),
		cs.Votes2.Copy(),
	}
}

// ToProto gets the protobuf representaton of the casper slashing.
func (cs *CasperSlashing) ToProto() *pb.CasperSlashing {
	return &pb.CasperSlashing{
		Vote0: cs.Votes1.ToProto(),
		Vote1: cs.Votes2.ToProto(),
	}
}

// CasperSlashingFromProto returns the casper slashing from the protobuf representation.
func CasperSlashingFromProto(slashing *pb.CasperSlashing) (*CasperSlashing, error) {
	votes1, err := SlashableVoteDataFromProto(slashing.Vote0)
	if err != nil {
		return nil, err
	}
	votes2, err := SlashableVoteDataFromProto(slashing.Vote1)
	if err != nil {
		return nil, err
	}
	return &CasperSlashing{
		Votes1: *votes1,
		Votes2: *votes2,
	}, nil
}
