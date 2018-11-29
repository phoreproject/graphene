package transaction

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/bls"
	pb "github.com/phoreproject/synapse/pb"
)

// Attestation is a signed attestation of a shard block.
type Attestation struct {
	Slot                uint64
	ShardID             uint64
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
	// Slot number
	Slot uint64
	// Shard number
	Shard uint64
	// CYCLE_LENGTH parent hashes
	ParentHashes []chainhash.Hash
	// Shard block hash
	ShardBlockHash chainhash.Hash
	// Last crosslink hash
	LastCrosslinkHash chainhash.Hash
	// Root of data between last hash and this one
	ShardBlockCombinedDataRoot chainhash.Hash
	// Slot of last justified beacon block referenced in the attestation
	JustifiedSlot uint64
}

// AttestationRecord is a signed attestation of a shard block.
type AttestationRecord struct {
	// Signed data
	Data AttestationSignedData
	// Attester participation bitfield
	AttesterBitfield []uint8
	// Proof of custody bitfield
	PoCBitfield []uint8
	// BLS aggregate signature
	// TODO: replace the type with BLS
	AggregateSig int
}

// NewAttestationRecordFromProto gets a new attestation from a protobuf attestation message.
func NewAttestationRecordFromProto(att *pb.AttestationRecord) (*AttestationRecord, error) {
	var data AttestationSignedData
	data.FromProto(att.Data)
	return &AttestationRecord{
		Data:             data,
		AttesterBitfield: att.AttesterBitfield[:],
		PoCBitfield:      att.PoCBitfield[:],
		AggregateSig:     0,
	}, nil
}

// ToProto gets the protobuf representation of the attestation
func (a *AttestationRecord) ToProto() *pb.AttestationRecord {
	var att pb.AttestationRecord
	att.Data = a.Data.ToProto()
	att.AttesterBitfield = a.AttesterBitfield[:]
	att.PoCBitfield = a.PoCBitfield[:]
	//att.AggregateSig = a.AggregateSig
	return &att
}

// ProcessedAttestation is the processed attestation
type ProcessedAttestation struct {
	// Signed data
	Data AttestationSignedData
	// Attester participation bitfield
	AttesterBitfield []uint8
	// Proof of custody bitfield
	PoCBitfield []uint8
	// Slot in which it was included
	SlotIncluded uint64
}

// ToProto gets the protobuf representation of the data.
func (a *AttestationSignedData) ToProto() *pb.AttestationSignedData {
	parentHashes := make([][]byte, len(a.ParentHashes))
	for i := range parentHashes {
		parentHashes[i] = a.ParentHashes[i][:]
	}

	return &pb.AttestationSignedData{
		Slot:           a.Slot,
		Shard:          a.Shard,
		ShardBlockHash: a.ShardBlockHash[:],
		ParentHashes:   parentHashes,
		JustifiedSlot:  a.JustifiedSlot,
	}
}

// FromProto constructs from proto
func (a *AttestationSignedData) FromProto(att *pb.AttestationSignedData) {
	a.ParentHashes = make([]chainhash.Hash, len(att.ParentHashes))
	for i := range att.ParentHashes {
		a.ParentHashes[i].SetBytes(att.ParentHashes[i])
	}
	a.Slot = att.Slot
	a.Shard = att.Shard
	a.ShardBlockHash.SetBytes(att.ShardBlockHash)
	a.JustifiedSlot = att.JustifiedSlot
}
