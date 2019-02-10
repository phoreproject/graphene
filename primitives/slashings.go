package primitives

import (
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
	copy(newPS.ProposalSignature1, ps.ProposalSignature1[:])
	copy(newPS.ProposalSignature2, ps.ProposalSignature2[:])
	return newPS
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
		AggregateSignaturePoC0Indices: svd.AggregateSignaturePoC0Indices[:],
		AggregateSignaturePoC1Indices: svd.AggregateSignaturePoC1Indices[:],
		Data:                          svd.Data.Copy(),
		AggregateSignature:            svd.AggregateSignature,
	}
}

// ToProto returns the protobuf representation of the slashable vote data.
func (svd *SlashableVoteData) ToProto() *pb.SlashableVoteData {
	newSvd := &pb.SlashableVoteData{
		AggregateSignaturePoC0Indices: svd.AggregateSignaturePoC0Indices,
		AggregateSignaturePoC1Indices: svd.AggregateSignaturePoC1Indices,
		Data:                          svd.Data.ToProto(),
	}
	copy(newSvd.AggregateSignature, svd.AggregateSignature[:])
	return newSvd
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
