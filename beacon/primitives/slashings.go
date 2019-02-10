package primitives

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
