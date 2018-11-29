package transaction

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/bls"
	pb "github.com/phoreproject/synapse/pb"
)

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
	AggregateSig bls.Signature
}

// NewAttestationRecordFromProto gets a new attestation from a protobuf attestation message.
func NewAttestationRecordFromProto(att *pb.AttestationRecord) (*AttestationRecord, error) {
	var data AttestationSignedData
	data.FromProto(att.Data)
	return &AttestationRecord{
		Data:             data,
		AttesterBitfield: att.AttesterBitfield[:],
		PoCBitfield:      att.PoCBitfield[:],
		AggregateSig:     bls.Signature{},
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
