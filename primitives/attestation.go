package primitives

import (
	"errors"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
)

// AttestationData is the part of the attestation that is signed.
type AttestationData struct {
	Slot uint64

	// FFG vote
	BeaconBlockHash chainhash.Hash

	SourceEpoch uint64
	SourceHash  chainhash.Hash
	TargetEpoch uint64
	TargetHash  chainhash.Hash

	// Shard number
	Shard               uint64
	LatestCrosslinkHash chainhash.Hash
	ShardBlockHash      chainhash.Hash
}

// Equals checks if this attestation data is equal to another.
func (a *AttestationData) Equals(other *AttestationData) bool {
	return a.BeaconBlockHash.IsEqual(&other.BeaconBlockHash) &&
		a.SourceEpoch == other.SourceEpoch &&
		a.SourceHash.IsEqual(&other.SourceHash) &&
		a.TargetEpoch == other.TargetEpoch &&
		a.TargetHash.IsEqual(&other.TargetHash) &&
		a.Shard == other.Shard &&
		a.LatestCrosslinkHash.IsEqual(&other.LatestCrosslinkHash) &&
		a.ShardBlockHash.IsEqual(&other.ShardBlockHash) &&
		a.Slot == other.Slot
}

// Copy returns a copy of the data.
func (a *AttestationData) Copy() AttestationData {
	return *a
}

// AttestationDataFromProto converts the protobuf representation to an attestationdata
// item.
func AttestationDataFromProto(att *pb.AttestationData) (*AttestationData, error) {
	a := &AttestationData{
		SourceEpoch: att.SourceEpoch,
		TargetEpoch: att.TargetEpoch,
		Shard:       att.Shard,
		Slot:        att.Slot,
	}

	if err := a.BeaconBlockHash.SetBytes(att.BeaconBlockHash); err != nil {
		return nil, err
	}
	if err := a.SourceHash.SetBytes(att.SourceHash); err != nil {
		return nil, err
	}
	if err := a.TargetHash.SetBytes(att.TargetHash); err != nil {
		return nil, err
	}
	if err := a.LatestCrosslinkHash.SetBytes(att.LatestCrosslinkHash); err != nil {
		return nil, err
	}
	if err := a.ShardBlockHash.SetBytes(att.ShardBlockHash); err != nil {
		return nil, err
	}

	return a, nil
}

// ToProto converts the attestation to protobuf form.
func (a AttestationData) ToProto() *pb.AttestationData {
	return &pb.AttestationData{
		Slot:            a.Slot,
		BeaconBlockHash: a.BeaconBlockHash[:],

		SourceEpoch: a.SourceEpoch,
		SourceHash:  a.SourceHash[:],
		TargetEpoch: a.TargetEpoch,
		TargetHash:  a.TargetHash[:],

		Shard:               a.Shard,
		ShardBlockHash:      a.ShardBlockHash[:],
		LatestCrosslinkHash: a.LatestCrosslinkHash[:],
	}
}

// AttestationDataAndCustodyBit is an attestation data and custody bit combined.
type AttestationDataAndCustodyBit struct {
	Data   AttestationData
	PoCBit bool
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
	newAtt := Attestation{
		Data: a.Data.Copy(),
	}

	copy(newAtt.AggregateSig[:], a.AggregateSig[:])
	newAtt.CustodyBitfield = append([]uint8{}, a.CustodyBitfield...)
	newAtt.ParticipationBitfield = append([]uint8{}, a.ParticipationBitfield...)

	return newAtt
}

// AttestationFromProto gets a new attestation from a protobuf attestation message.
func AttestationFromProto(att *pb.Attestation) (*Attestation, error) {
	// att can be nil if invalid message is sent.
	if att == nil || att.Data == nil {
		return nil, errors.New("attestation can't be nil")
	}
	data, err := AttestationDataFromProto(att.Data)
	if err != nil {
		return nil, err
	}
	a := &Attestation{
		Data:                  *data,
		ParticipationBitfield: att.ParticipationBitfield[:],
		CustodyBitfield:       att.CustodyBitfield[:],
	}
	if len(att.AggregateSig) > 48 {
		return nil, errors.New("aggregateSig was not less than 48 bytes")
	}
	copy(a.AggregateSig[:], att.AggregateSig)
	return a, nil
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
	InclusionDelay        uint64
	ProposerIndex         uint32
}

// Copy copies a pending attestation
func (pa *PendingAttestation) Copy() PendingAttestation {
	newPa := *pa
	newPa.ParticipationBitfield = append([]uint8{}, pa.ParticipationBitfield...)
	newPa.CustodyBitfield = append([]uint8{}, pa.CustodyBitfield...)
	return newPa
}

// ToProto returns a protobuf representation of the pending attestation.
func (pa *PendingAttestation) ToProto() *pb.PendingAttestation {
	paProto := &pb.PendingAttestation{}
	paProto.CustodyBitfield = pa.CustodyBitfield
	paProto.Data = pa.Data.ToProto()
	paProto.ParticipationBitfield = pa.ParticipationBitfield
	paProto.InclusionDelay = pa.InclusionDelay
	paProto.ProposerIndex = pa.ProposerIndex
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
		InclusionDelay:        pa.InclusionDelay,
		ProposerIndex:         pa.ProposerIndex,
		Data:                  *data,
	}, nil
}
