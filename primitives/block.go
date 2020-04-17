package primitives

import (
	"errors"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
)

// Block represents a single beacon chain block.
type Block struct {
	BlockHeader BlockHeader
	BlockBody   BlockBody
}

// Copy returns a copy of the block.
func (b *Block) Copy() Block {
	return Block{
		b.BlockHeader.Copy(),
		b.BlockBody.Copy(),
	}
}

// ToProto gets the protobuf representation of the block.
func (b *Block) ToProto() *pb.Block {
	return &pb.Block{
		Header: b.BlockHeader.ToProto(),
		Body:   b.BlockBody.ToProto(),
	}
}

// BlockFromProto returns a block from the protobuf representation.
func BlockFromProto(bl *pb.Block) (*Block, error) {
	header, err := BlockHeaderFromProto(bl.Header)
	if err != nil {
		return nil, err
	}

	body, err := BlockBodyFromProto(bl.Body)
	if err != nil {
		return nil, err
	}

	return &Block{
		BlockHeader: *header,
		BlockBody:   *body,
	}, nil
}

// BlockHeader is the header of the block.
type BlockHeader struct {
	SlotNumber     uint64
	ParentRoot     chainhash.Hash
	StateRoot      chainhash.Hash
	ValidatorIndex uint32
	RandaoReveal   [48]byte
	Signature      [48]byte
}

// Copy returns a copy of the block header.
func (bh *BlockHeader) Copy() BlockHeader {
	return *bh
}

// ToProto converts to block header to protobuf form.
func (bh *BlockHeader) ToProto() *pb.BlockHeader {
	return &pb.BlockHeader{
		SlotNumber:     bh.SlotNumber,
		ValidatorIndex: bh.ValidatorIndex,
		ParentRoot:     bh.ParentRoot[:],
		StateRoot:      bh.StateRoot[:],
		RandaoReveal:   bh.RandaoReveal[:],
		Signature:      bh.Signature[:],
	}
}

// BlockHeaderFromProto converts a protobuf representation of a block header to a block header.
func BlockHeaderFromProto(header *pb.BlockHeader) (*BlockHeader, error) {
	if len(header.RandaoReveal) > 48 {
		return nil, errors.New("randaoReveal should be 48 bytes long")
	}
	if len(header.Signature) > 48 {
		return nil, errors.New("signature should be 48 bytes long")
	}
	newHeader := &BlockHeader{
		SlotNumber:     header.SlotNumber,
		ValidatorIndex: header.ValidatorIndex,
	}

	copy(newHeader.RandaoReveal[:], header.RandaoReveal)
	copy(newHeader.Signature[:], header.Signature)

	err := newHeader.StateRoot.SetBytes(header.StateRoot)
	if err != nil {
		return nil, err
	}

	err = newHeader.ParentRoot.SetBytes(header.ParentRoot)
	if err != nil {
		return nil, err
	}

	return newHeader, nil
}

// BlockBody contains the beacon actions that happened this block.
type BlockBody struct {
	Attestations      []Attestation
	ProposerSlashings []ProposerSlashing
	CasperSlashings   []CasperSlashing
	Deposits          []Deposit
	Exits             []Exit
	Votes             []AggregatedVote
}

// Copy returns a copy of the block body.
func (bb *BlockBody) Copy() BlockBody {
	newAttestations := make([]Attestation, len(bb.Attestations))
	newProposerSlashings := make([]ProposerSlashing, len(bb.ProposerSlashings))
	newCasperSlashings := make([]CasperSlashing, len(bb.CasperSlashings))
	newDeposits := make([]Deposit, len(bb.Deposits))
	newExits := make([]Exit, len(bb.Exits))
	newVotes := make([]AggregatedVote, len(bb.Votes))

	for i := range bb.Attestations {
		newAttestations[i] = bb.Attestations[i].Copy()
	}

	for i := range bb.ProposerSlashings {
		newProposerSlashings[i] = bb.ProposerSlashings[i].Copy()
	}

	for i := range bb.CasperSlashings {
		newCasperSlashings[i] = bb.CasperSlashings[i].Copy()
	}

	for i := range bb.Deposits {
		newDeposits[i] = bb.Deposits[i].Copy()
	}

	for i := range bb.Exits {
		newExits[i] = bb.Exits[i].Copy()
	}

	for i := range bb.Votes {
		newVotes[i] = bb.Votes[i].Copy()
	}

	return BlockBody{
		Attestations:      newAttestations,
		ProposerSlashings: newProposerSlashings,
		CasperSlashings:   newCasperSlashings,
		Deposits:          newDeposits,
		Exits:             newExits,
		Votes:             newVotes,
	}
}

// ToProto converts the block body to protobuf form
func (bb *BlockBody) ToProto() *pb.BlockBody {
	atts := make([]*pb.Attestation, len(bb.Attestations))
	for i := range atts {
		atts[i] = bb.Attestations[i].ToProto()
	}
	ps := make([]*pb.ProposerSlashing, len(bb.ProposerSlashings))
	for i := range ps {
		ps[i] = bb.ProposerSlashings[i].ToProto()
	}
	cs := make([]*pb.CasperSlashing, len(bb.CasperSlashings))
	for i := range cs {
		cs[i] = bb.CasperSlashings[i].ToProto()
	}
	ds := make([]*pb.Deposit, len(bb.Deposits))
	for i := range ds {
		ds[i] = bb.Deposits[i].ToProto()
	}
	ex := make([]*pb.Exit, len(bb.Exits))
	for i := range ex {
		ex[i] = bb.Exits[i].ToProto()
	}
	vs := make([]*pb.AggregatedVote, len(bb.Votes))
	for i := range vs {
		vs[i] = bb.Votes[i].ToProto()
	}
	return &pb.BlockBody{
		Attestations:      atts,
		ProposerSlashings: ps,
		CasperSlashings:   cs,
		Deposits:          ds,
		Exits:             ex,
		Votes:             vs,
	}
}

// BlockBodyFromProto converts a protobuf representation of a block body to a block body.
func BlockBodyFromProto(body *pb.BlockBody) (*BlockBody, error) {
	atts := make([]Attestation, len(body.Attestations))
	casperSlashings := make([]CasperSlashing, len(body.CasperSlashings))
	proposerSlashings := make([]ProposerSlashing, len(body.ProposerSlashings))
	deposits := make([]Deposit, len(body.Deposits))
	exits := make([]Exit, len(body.Exits))
	votes := make([]AggregatedVote, len(body.Votes))

	for i := range atts {
		a, err := AttestationFromProto(body.Attestations[i])
		if err != nil {
			return nil, err
		}
		atts[i] = *a
	}

	for i := range casperSlashings {
		cs, err := CasperSlashingFromProto(body.CasperSlashings[i])
		if err != nil {
			return nil, err
		}
		casperSlashings[i] = *cs
	}

	for i := range proposerSlashings {
		ps, err := ProposerSlashingFromProto(body.ProposerSlashings[i])
		if err != nil {
			return nil, err
		}
		proposerSlashings[i] = *ps
	}

	for i := range deposits {
		ds, err := DepositFromProto(body.Deposits[i])
		if err != nil {
			return nil, err
		}
		deposits[i] = *ds
	}

	for i := range exits {
		ex, err := ExitFromProto(body.Exits[i])
		if err != nil {
			return nil, err
		}
		exits[i] = *ex
	}

	for i := range votes {
		v, err := AggregatedVoteFromProto(body.Votes[i])
		if err != nil {
			return nil, err
		}
		votes[i] = *v
	}

	return &BlockBody{
		Attestations:      atts,
		CasperSlashings:   casperSlashings,
		ProposerSlashings: proposerSlashings,
		Deposits:          deposits,
		Exits:             exits,
		Votes:             votes,
	}, nil
}
