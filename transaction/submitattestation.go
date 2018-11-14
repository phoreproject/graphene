package transaction

import (
	"bytes"
	"encoding/binary"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/bls"
	pb "github.com/phoreproject/synapse/pb"
)

// Attestation is a signed attestation of a shard block.
type Attestation struct {
	Slot                uint64
	ShardID             uint32
	JustifiedSlot       uint64
	JustifiedBlockHash  chainhash.Hash
	ShardBlockHash      chainhash.Hash
	ObliqueParentHashes []chainhash.Hash
	AttesterBitField    []byte
	AggregateSignature  bls.Signature
}

// NewAttestationFromProto gets a new attestation from a protobuf attestation
// message.
func NewAttestationFromProto(att *pb.Attestation) (*Attestation, error) {
	justifiedBlockHash, err := chainhash.NewHash(att.JustifiedBlockHash)
	if err != nil {
		return nil, err
	}

	shardBlockHash, err := chainhash.NewHash(att.ShardBlockHash)
	if err != nil {
		return nil, err
	}

	obliqueParentHashes := make([]chainhash.Hash, len(att.ObliqueParentHashes))
	for i := range obliqueParentHashes {
		h, err := chainhash.NewHash(att.ObliqueParentHashes[i])
		if err != nil {
			return nil, err
		}
		obliqueParentHashes[i] = *h
	}

	s, err := bls.DeserializeSignature(att.AggregateSignature)
	if err != nil {
		return nil, err
	}

	return &Attestation{
		Slot:                att.Slot,
		ShardID:             att.ShardID,
		JustifiedSlot:       att.JustifiedSlot,
		ShardBlockHash:      *shardBlockHash,
		JustifiedBlockHash:  *justifiedBlockHash,
		ObliqueParentHashes: obliqueParentHashes,
		AttesterBitField:    att.AttesterBitField,
		AggregateSignature:  s,
	}, nil
}

// ToProto gets the protobuf representation of the attestation
func (a *Attestation) ToProto() *pb.Attestation {
	obliqueParentHashes := make([][]byte, len(a.ObliqueParentHashes))
	for i := range obliqueParentHashes {
		obliqueParentHashes[i] = a.ObliqueParentHashes[i][:]
	}
	return &pb.Attestation{
		Slot:                a.Slot,
		ShardID:             a.ShardID,
		JustifiedSlot:       a.JustifiedSlot,
		JustifiedBlockHash:  a.JustifiedBlockHash[:],
		ObliqueParentHashes: obliqueParentHashes,
		AttesterBitField:    a.AttesterBitField,
		AggregateSignature:  a.AggregateSignature.Serialize(),
	}
}

// AttestationSignedData is the part of the attestation that is signed.
type AttestationSignedData struct {
	Version        uint32
	Slot           uint64
	Shard          uint32
	ParentHashes   []chainhash.Hash
	ShardBlockHash chainhash.Hash
	JustifiedSlot  uint64
}

// Serialize gets the binary representation of the signed data
func (a *AttestationSignedData) Serialize() []byte {
	b := bytes.NewBuffer([]byte{})
	binary.Write(b, binary.BigEndian, *a)
	return b.Bytes()
}
