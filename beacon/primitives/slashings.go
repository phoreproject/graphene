package primitives

import (
	"github.com/phoreproject/synapse/bls"
)

// ProposerSlashing is a slashing request for a proposal violation.
type ProposerSlashing struct {
	ProposerIndex      uint32
	ProposalData1      ProposalSignedData
	ProposalSignature1 bls.Signature
	ProposalData2      ProposalSignedData
	ProposalSignature2 bls.Signature
}

// Copy returns a copy of the proposer slashing.
func (ps *ProposerSlashing) Copy() ProposerSlashing {
	newSignature1 := ps.ProposalSignature1.Copy()
	newSignature2 := ps.ProposalSignature1.Copy()
	newProposerSlashing := *ps
	newProposerSlashing.ProposalSignature1 = *newSignature1
	newProposerSlashing.ProposalSignature2 = *newSignature2
	return newProposerSlashing
}

// SlashableVoteData is the vote data that should be slashed.
type SlashableVoteData struct {
	AggregateSignaturePoC0Indices []uint32
	AggregateSignaturePoC1Indices []uint32
	Data                          AttestationData
	AggregateSignature            bls.Signature
}

// Copy returns a copy of the slashable vote data.
func (svd *SlashableVoteData) Copy() SlashableVoteData {
	newSignature := svd.AggregateSignature.Copy()
	return SlashableVoteData{
		AggregateSignaturePoC0Indices: svd.AggregateSignaturePoC0Indices[:],
		AggregateSignaturePoC1Indices: svd.AggregateSignaturePoC1Indices[:],
		Data:                          svd.Data.Copy(),
		AggregateSignature:            *newSignature,
	}
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
