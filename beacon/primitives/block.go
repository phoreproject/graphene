package primitives

import (
	"github.com/golang/protobuf/proto"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/bls"
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

// Hash gets the hash of the block header
func (b *Block) Hash() chainhash.Hash {
	m, _ := proto.Marshal(b.ToProto())
	return chainhash.HashH(m)
}
