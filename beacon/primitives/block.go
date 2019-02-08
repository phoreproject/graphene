package primitives

import (
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
)

// Block represents a single beacon chain block.
type Block struct {
	BlockHeader
	BlockBody
}

// Copy returns a copy of the block.
func (b *Block) Copy() Block {
	return Block{
		b.BlockHeader.Copy(),
		b.BlockBody.Copy(),
	}
}

// BlockHeader is the header of the block.
type BlockHeader struct {
	SlotNumber   uint64
	ParentRoot   chainhash.Hash
	StateRoot    chainhash.Hash
	RandaoReveal chainhash.Hash
	Signature    bls.Signature
}

// EncodeSSZ implements Encodable
func (bh BlockHeader) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, bh.SlotNumber); err != nil {
		return err
	}
	if err := ssz.Encode(writer, bh.ParentRoot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, bh.StateRoot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, bh.RandaoReveal); err != nil {
		return err
	}
	if err := ssz.Encode(writer, bh.Signature); err != nil {
		return err
	}
	return nil
}

// EncodeSSZSize implements Encodable
func (bh BlockHeader) EncodeSSZSize() (uint32, error) {
	var sizeOfslotNumber, sizeOfparentRoot, sizeOfstateRoot, sizeOfrandaoReveal, sizeOfsignature uint32
	var err error
	if sizeOfslotNumber, err = ssz.EncodeSize(bh.SlotNumber); err != nil {
		return 0, err
	}
	if sizeOfparentRoot, err = ssz.EncodeSize(bh.ParentRoot); err != nil {
		return 0, err
	}
	if sizeOfstateRoot, err = ssz.EncodeSize(bh.StateRoot); err != nil {
		return 0, err
	}
	if sizeOfrandaoReveal, err = ssz.EncodeSize(bh.RandaoReveal); err != nil {
		return 0, err
	}
	if sizeOfsignature, err = ssz.EncodeSize(bh.Signature); err != nil {
		return 0, err
	}
	return sizeOfslotNumber + sizeOfparentRoot + sizeOfstateRoot + sizeOfrandaoReveal + sizeOfsignature, nil
}

// DecodeSSZ implements Decodable
func (bh BlockHeader) DecodeSSZ(reader io.Reader) error {
	if err := ssz.Decode(reader, &bh.SlotNumber); err != nil {
		return err
	}
	bh.ParentRoot = chainhash.Hash{}
	if err := ssz.Decode(reader, &bh.ParentRoot); err != nil {
		return err
	}
	bh.StateRoot = chainhash.Hash{}
	if err := ssz.Decode(reader, &bh.StateRoot); err != nil {
		return err
	}
	bh.RandaoReveal = chainhash.Hash{}
	if err := ssz.Decode(reader, &bh.RandaoReveal); err != nil {
		return err
	}
	bh.Signature = bls.Signature{}
	if err := ssz.Decode(reader, &bh.Signature); err != nil {
		return err
	}
	return nil
}

// Copy returns a copy of the block header.
func (bh *BlockHeader) Copy() BlockHeader {
	newHeader := *bh
	newSignature := bh.Signature.Copy()
	newHeader.Signature = *newSignature
	return newHeader
}

// ToProto converts to block header to protobuf form.
func (bh *BlockHeader) ToProto() *pb.BlockHeader {
	return &pb.BlockHeader{
		SlotNumber:   bh.SlotNumber,
		ParentRoot:   bh.ParentRoot[:],
		StateRoot:    bh.StateRoot[:],
		RandaoReveal: bh.RandaoReveal[:],
		Signature:    bh.Signature.Serialize(),
	}
}

// BlockBody contains the beacon actions that happened this block.
type BlockBody struct {
	Attestations      []Attestation
	ProposerSlashings []ProposerSlashing
	CasperSlashings   []CasperSlashing
	Deposits          []Deposit
	Exits             []Exit
}

// Copy returns a copy of the block body.
func (bb *BlockBody) Copy() BlockBody {
	newAttestations := make([]Attestation, len(bb.Attestations))
	newProposerSlashings := make([]ProposerSlashing, len(bb.ProposerSlashings))
	newCasperSlashings := make([]CasperSlashing, len(bb.CasperSlashings))
	newDeposits := make([]Deposit, len(bb.Deposits))
	newExits := make([]Exit, len(bb.Exits))

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

	return BlockBody{
		Attestations:      newAttestations,
		ProposerSlashings: newProposerSlashings,
		CasperSlashings:   newCasperSlashings,
		Deposits:          newDeposits,
		Exits:             newExits,
	}
}

// TreeHashSSZ gets the hash of the block header
func (b *Block) TreeHashSSZ() (chainhash.Hash, error) {
	m, _ := proto.Marshal(b.ToProto())
	return chainhash.HashH(m), nil
}
