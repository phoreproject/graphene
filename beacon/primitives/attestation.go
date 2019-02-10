package primitives

import (
	"github.com/phoreproject/synapse/chainhash"
	pb "github.com/phoreproject/synapse/pb"
)

// AttestationData is the part of the attestation that is signed.
type AttestationData struct {
	// Slot number
	Slot uint64
	// Shard number
	Shard               uint64
	BeaconBlockHash     chainhash.Hash
	EpochBoundaryHash   chainhash.Hash
	ShardBlockHash      chainhash.Hash
	LatestCrosslinkHash chainhash.Hash
	JustifiedSlot       uint64
	JustifiedBlockHash  chainhash.Hash
}

// Equals checks if this attestation data is equal to another.
func (a *AttestationData) Equals(other *AttestationData) bool {
	return a.Slot == other.Slot &&
		a.Shard == other.Shard &&
		a.BeaconBlockHash.IsEqual(&other.BeaconBlockHash) &&
		a.EpochBoundaryHash.IsEqual(&other.EpochBoundaryHash) &&
		a.ShardBlockHash.IsEqual(&other.ShardBlockHash) &&
		a.LatestCrosslinkHash.IsEqual(&other.LatestCrosslinkHash) &&
		a.JustifiedSlot == other.JustifiedSlot &&
		a.JustifiedBlockHash.IsEqual(&other.JustifiedBlockHash)
}

// Copy returns a copy of the data.
func (a *AttestationData) Copy() AttestationData {
	return *a
}

// AttestationDataAndCustodyBit is an attestation data and custody bit combined.
type AttestationDataAndCustodyBit struct {
	Data   AttestationData
	PoCBit bool
}

// AttestationDataFromProto converts the protobuf representation to an attestationdata
// item.
func AttestationDataFromProto(att *pb.AttestationData) (*AttestationData, error) {
	a := &AttestationData{}
	a.Slot = att.Slot
	a.Shard = att.Shard
	err := a.ShardBlockHash.SetBytes(att.ShardBlockHash)
	if err != nil {
		return nil, err
	}
	err = a.BeaconBlockHash.SetBytes(att.BeaconBlockHash)
	if err != nil {
		return nil, err
	}
	err = a.EpochBoundaryHash.SetBytes(att.EpochBoundaryHash)
	if err != nil {
		return nil, err
	}
	err = a.LatestCrosslinkHash.SetBytes(att.LatestCrosslinkHash)
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

// ToProto converts the attestation to protobuf form.
func (a AttestationData) ToProto() *pb.AttestationData {
	return &pb.AttestationData{
		Slot:                a.Slot,
		Shard:               a.Shard,
		BeaconBlockHash:     a.BeaconBlockHash[:],
		EpochBoundaryHash:   a.EpochBoundaryHash[:],
		ShardBlockHash:      a.ShardBlockHash[:],
		LatestCrosslinkHash: a.LatestCrosslinkHash[:],
		JustifiedSlot:       a.JustifiedSlot,
		JustifiedBlockHash:  a.JustifiedBlockHash[:],
	}
}

// Attestation is a signed attestation of a shard block.
type Attestation struct {
	// Signed data
	Data AttestationData
	// Attester participation bitfield
	ParticipationBitfield []uint8
	// Proof of custody bitfield
	CustodyBitfield []uint8
	// BLS aggregate signature
	AggregateSig [48]byte
}

// Copy returns a copy of the attestation
func (a *Attestation) Copy() Attestation {
	sig := [48]byte{}
	copy(sig[:], a.AggregateSig[:])
	return Attestation{
		Data:                  a.Data.Copy(),
		ParticipationBitfield: a.ParticipationBitfield[:],
		CustodyBitfield:       a.CustodyBitfield[:],
		AggregateSig:          sig,
	}
}

// AttestationFromProto gets a new attestation from a protobuf attestation message.
func AttestationFromProto(att *pb.Attestation) (*Attestation, error) {
	data, err := AttestationDataFromProto(att.Data)
	if err != nil {
		return nil, err
	}
	return &Attestation{
		Data:                  *data,
		ParticipationBitfield: att.ParticipationBitfield[:],
		CustodyBitfield:       att.CustodyBitfield[:],
		AggregateSig:          [48]byte{},
	}, nil
}

// ToProto gets the protobuf representation of the attestation
func (a *Attestation) ToProto() *pb.Attestation {
	att := &pb.Attestation{}
	att.Data = a.Data.ToProto()
	att.ParticipationBitfield = a.ParticipationBitfield[:]
	att.CustodyBitfield = a.CustodyBitfield[:]
	att.AggregateSig = a.AggregateSig[:]
	return att
}

// PendingAttestation is an attestation waiting to be included.
type PendingAttestation struct {
	Data                  AttestationData
	ParticipationBitfield []byte
	CustodyBitfield       []byte
	SlotIncluded          uint64
}

// Copy copies a pending attestation
func (pa *PendingAttestation) Copy() PendingAttestation {
	newPa := *pa
	copy(newPa.ParticipationBitfield, pa.ParticipationBitfield)
	copy(newPa.CustodyBitfield, pa.CustodyBitfield)
	return newPa
}

// ToProto returns a protobuf representation of the pending attestation.
func (pa *PendingAttestation) ToProto() *pb.PendingAttestation {
	paProto := &pb.PendingAttestation{}
	paProto.CustodyBitfield = pa.CustodyBitfield
	paProto.Data = pa.Data.ToProto()
	paProto.ParticipationBitfield = pa.ParticipationBitfield
	paProto.SlotIncluded = pa.SlotIncluded
	return paProto
}

// PendingAttestationFromProto converts a protobuf attestation to a
// pending attestation.
func PendingAttestationFromProto(pa *pb.PendingAttestation) (*PendingAttestation, error) {
	data, err := AttestationDataFromProto(pa.Data)
	if err != nil {
		return nil, err
	}
	return &PendingAttestation{
		CustodyBitfield:       pa.CustodyBitfield,
		ParticipationBitfield: pa.ParticipationBitfield,
		SlotIncluded:          pa.SlotIncluded,
		Data:                  *data,
	}, nil
}
