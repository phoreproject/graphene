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
	// Hash of last justified beacon block
	JustifiedBlockHash chainhash.Hash
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
	data, err := AttestationSignedDataFromProto(att.Data)
	if err != nil {
		return nil, err
	}
	return &AttestationRecord{
		Data:             *data,
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

// ToProto converts the attestation signed data into protobuf form.
func (asd *AttestationSignedData) ToProto() *pb.AttestationSignedData {
	parentHashes := make([][]byte, len(asd.ParentHashes))
	for i := range parentHashes {
		parentHashes[i] = asd.ParentHashes[i][:]
	}

	return &pb.AttestationSignedData{
		Slot:                       asd.Slot,
		Shard:                      asd.Shard,
		ParentHashes:               parentHashes,
		ShardBlockHash:             asd.ShardBlockHash[:],
		LastCrosslinkHash:          asd.LastCrosslinkHash[:],
		ShardBlockCombinedDataRoot: asd.ShardBlockCombinedDataRoot[:],
		JustifiedSlot:              asd.JustifiedSlot,
		JustifiedBlockHash:         asd.JustifiedBlockHash[:],
	}
}

// AttestationSignedDataFromProto constructs an attestation signed data from
// the protobuf representation.
func AttestationSignedDataFromProto(att *pb.AttestationSignedData) (*AttestationSignedData, error) {
	a := &AttestationSignedData{}
	a.ParentHashes = make([]chainhash.Hash, len(att.ParentHashes))
	for i := range att.ParentHashes {
		a.ParentHashes[i].SetBytes(att.ParentHashes[i])
	}
	a.Slot = att.Slot
	a.Shard = att.Shard
	err := a.ShardBlockHash.SetBytes(att.ShardBlockHash)
	if err != nil {
		return nil, err
	}
	err = a.LastCrosslinkHash.SetBytes(att.LastCrosslinkHash)
	if err != nil {
		return nil, err
	}
	err = a.ShardBlockCombinedDataRoot.SetBytes(att.ShardBlockCombinedDataRoot)
	if err != nil {
		return nil, err
	}
	err = a.JustifiedBlockHash.SetBytes(att.JustifiedBlockHash)
	if err != nil {
		return nil, err
	}
	a.JustifiedSlot = att.JustifiedSlot
	return a, nil
}
