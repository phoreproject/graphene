package primitives

import (
	"io"

	"github.com/phoreproject/synapse/bls"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

// ProposerSlashing is a slashing request for a proposal violation.
type ProposerSlashing struct {
	ProposerIndex      uint32
	ProposalData1      ProposalSignedData
	ProposalSignature1 bls.Signature
	ProposalData2      ProposalSignedData
	ProposalSignature2 bls.Signature
}

// EncodeSSZ implements Encodable
func (ps ProposerSlashing) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, ps.ProposerIndex); err != nil {
		return err
	}
	if err := ssz.Encode(writer, ps.ProposalData1); err != nil {
		return err
	}
	if err := ssz.Encode(writer, ps.ProposalSignature1); err != nil {
		return err
	}
	if err := ssz.Encode(writer, ps.ProposalData2); err != nil {
		return err
	}
	if err := ssz.Encode(writer, ps.ProposalSignature2); err != nil {
		return err
	}
	return nil
}

// EncodeSSZSize implements Encodable
func (ps ProposerSlashing) EncodeSSZSize() (uint32, error) {
	var sizeOfproposerIndex, sizeOfproposalData1, sizeOfproposalSignature1, sizeOfproposalData2, sizeOfproposalSignature2 uint32
	var err error
	if sizeOfproposerIndex, err = ssz.EncodeSize(ps.ProposerIndex); err != nil {
		return 0, err
	}
	if sizeOfproposalData1, err = ssz.EncodeSize(ps.ProposalData1); err != nil {
		return 0, err
	}
	if sizeOfproposalSignature1, err = ssz.EncodeSize(ps.ProposalSignature1); err != nil {
		return 0, err
	}
	if sizeOfproposalData2, err = ssz.EncodeSize(ps.ProposalData2); err != nil {
		return 0, err
	}
	if sizeOfproposalSignature2, err = ssz.EncodeSize(ps.ProposalSignature2); err != nil {
		return 0, err
	}
	return sizeOfproposerIndex + sizeOfproposalData1 + sizeOfproposalSignature1 + sizeOfproposalData2 + sizeOfproposalSignature2, nil
}

// DecodeSSZ implements Decodable
func (ps ProposerSlashing) DecodeSSZ(reader io.Reader) error {
	if err := ssz.Decode(reader, ps.ProposerIndex); err != nil {
		return err
	}
	ps.ProposalData1 = ProposalSignedData{}
	if err := ssz.Decode(reader, ps.ProposalData1); err != nil {
		return err
	}
	ps.ProposalSignature1 = bls.Signature{}
	if err := ssz.Decode(reader, ps.ProposalSignature1); err != nil {
		return err
	}
	ps.ProposalData2 = ProposalSignedData{}
	if err := ssz.Decode(reader, ps.ProposalData2); err != nil {
		return err
	}
	ps.ProposalSignature2 = bls.Signature{}
	if err := ssz.Decode(reader, ps.ProposalSignature2); err != nil {
		return err
	}
	return nil
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

// EncodeSSZ implements Encodable
func (svd SlashableVoteData) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, svd.AggregateSignaturePoC0Indices); err != nil {
		return err
	}
	if err := ssz.Encode(writer, svd.AggregateSignaturePoC1Indices); err != nil {
		return err
	}
	if err := ssz.Encode(writer, svd.Data); err != nil {
		return err
	}
	if err := ssz.Encode(writer, svd.AggregateSignature); err != nil {
		return err
	}
	return nil
}

// EncodeSSZSize implements Encodable
func (svd SlashableVoteData) EncodeSSZSize() (uint32, error) {
	var sizeOfaggregateSignaturePoC0Indices, sizeOfaggregateSignaturePoC1Indices, sizeOfdata, sizeOfaggregateSignature uint32
	var err error
	if sizeOfaggregateSignaturePoC0Indices, err = ssz.EncodeSize(svd.AggregateSignaturePoC0Indices); err != nil {
		return 0, err
	}
	if sizeOfaggregateSignaturePoC1Indices, err = ssz.EncodeSize(svd.AggregateSignaturePoC1Indices); err != nil {
		return 0, err
	}
	if sizeOfdata, err = ssz.EncodeSize(svd.Data); err != nil {
		return 0, err
	}
	if sizeOfaggregateSignature, err = ssz.EncodeSize(svd.AggregateSignature); err != nil {
		return 0, err
	}
	return sizeOfaggregateSignaturePoC0Indices + sizeOfaggregateSignaturePoC1Indices + sizeOfdata + sizeOfaggregateSignature, nil
}

// DecodeSSZ implements Decodable
func (svd SlashableVoteData) DecodeSSZ(reader io.Reader) error {
	svd.AggregateSignaturePoC0Indices = []uint32{}
	if err := ssz.Decode(reader, svd.AggregateSignaturePoC0Indices); err != nil {
		return err
	}
	svd.AggregateSignaturePoC1Indices = []uint32{}
	if err := ssz.Decode(reader, svd.AggregateSignaturePoC1Indices); err != nil {
		return err
	}
	svd.Data = AttestationData{}
	if err := ssz.Decode(reader, svd.Data); err != nil {
		return err
	}
	svd.AggregateSignature = bls.Signature{}
	if err := ssz.Decode(reader, svd.AggregateSignature); err != nil {
		return err
	}
	return nil
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

// EncodeSSZ implements Encodable
func (cs CasperSlashing) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, cs.Votes1); err != nil {
		return err
	}
	if err := ssz.Encode(writer, cs.Votes2); err != nil {
		return err
	}
	return nil
}

// EncodeSSZSize implements Encodable
func (cs CasperSlashing) EncodeSSZSize() (uint32, error) {
	var sizeOfvotes1, sizeOfvotes2 uint32
	var err error
	if sizeOfvotes1, err = ssz.EncodeSize(cs.Votes1); err != nil {
		return 0, err
	}
	if sizeOfvotes2, err = ssz.EncodeSize(cs.Votes2); err != nil {
		return 0, err
	}
	return sizeOfvotes1 + sizeOfvotes2, nil
}

// DecodeSSZ implements Decodable
func (cs CasperSlashing) DecodeSSZ(reader io.Reader) error {
	cs.Votes1 = SlashableVoteData{}
	if err := ssz.Decode(reader, cs.Votes1); err != nil {
		return err
	}
	cs.Votes2 = SlashableVoteData{}
	if err := ssz.Decode(reader, cs.Votes2); err != nil {
		return err
	}
	return nil
}

// Copy returns a copy of the casper slashing.
func (cs *CasperSlashing) Copy() CasperSlashing {
	return CasperSlashing{
		cs.Votes1.Copy(),
		cs.Votes2.Copy(),
	}
}
