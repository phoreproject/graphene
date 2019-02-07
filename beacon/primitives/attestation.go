package primitives

import (
	"io"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	pb "github.com/phoreproject/synapse/pb"
	"github.com/prysmaticlabs/prysm/shared/ssz"
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

// Copy returns a copy of the data.
func (a *AttestationData) Copy() AttestationData {
	return *a
}

// EncodeSSZ implements Encodable
func (a AttestationData) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, a.Slot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, a.Shard); err != nil {
		return err
	}
	if err := ssz.Encode(writer, a.BeaconBlockHash); err != nil {
		return err
	}
	if err := ssz.Encode(writer, a.EpochBoundaryHash); err != nil {
		return err
	}
	if err := ssz.Encode(writer, a.ShardBlockHash); err != nil {
		return err
	}
	if err := ssz.Encode(writer, a.LatestCrosslinkHash); err != nil {
		return err
	}
	if err := ssz.Encode(writer, a.JustifiedSlot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, a.JustifiedBlockHash); err != nil {
		return err
	}

	return nil
}

// EncodeSSZSize implements Encodable
func (a AttestationData) EncodeSSZSize() (uint32, error) {
	var sizeOfSlot, sizeOShard, sizeOfBeaconBlockHash, sizeOfEpochBoundaryHash,
		sizeOfShardBlockHash, sizeOfLatestCrosslinkHash, sizeOfJustifiedSlot, sizeOfJustifiedBlockHash uint32
	var err error
	if sizeOfSlot, err = ssz.EncodeSize(a.Slot); err != nil {
		return 0, err
	}
	if sizeOShard, err = ssz.EncodeSize(a.Shard); err != nil {
		return 0, err
	}
	if sizeOfBeaconBlockHash, err = ssz.EncodeSize(a.BeaconBlockHash); err != nil {
		return 0, err
	}
	if sizeOfEpochBoundaryHash, err = ssz.EncodeSize(a.EpochBoundaryHash); err != nil {
		return 0, err
	}
	if sizeOfShardBlockHash, err = ssz.EncodeSize(a.ShardBlockHash); err != nil {
		return 0, err
	}
	if sizeOfLatestCrosslinkHash, err = ssz.EncodeSize(a.LatestCrosslinkHash); err != nil {
		return 0, err
	}
	if sizeOfJustifiedSlot, err = ssz.EncodeSize(a.JustifiedSlot); err != nil {
		return 0, err
	}
	if sizeOfJustifiedBlockHash, err = ssz.EncodeSize(a.JustifiedBlockHash); err != nil {
		return 0, err
	}
	return sizeOfSlot + sizeOShard + sizeOfBeaconBlockHash + sizeOfEpochBoundaryHash +
		sizeOfShardBlockHash + sizeOfLatestCrosslinkHash + sizeOfJustifiedSlot + sizeOfJustifiedBlockHash, nil
}

// DecodeSSZ implements Decodable
func (a AttestationData) DecodeSSZ(reader io.Reader) error {
	if err := ssz.Decode(reader, a.Slot); err != nil {
		return err
	}
	if err := ssz.Decode(reader, a.Shard); err != nil {
		return err
	}
	a.BeaconBlockHash = chainhash.Hash{}
	if err := ssz.Decode(reader, a.BeaconBlockHash); err != nil {
		return err
	}
	a.EpochBoundaryHash = chainhash.Hash{}
	if err := ssz.Decode(reader, a.EpochBoundaryHash); err != nil {
		return err
	}
	a.ShardBlockHash = chainhash.Hash{}
	if err := ssz.Decode(reader, a.ShardBlockHash); err != nil {
		return err
	}
	a.LatestCrosslinkHash = chainhash.Hash{}
	if err := ssz.Decode(reader, a.LatestCrosslinkHash); err != nil {
		return err
	}
	if err := ssz.Decode(reader, a.JustifiedSlot); err != nil {
		return err
	}
	a.JustifiedBlockHash = chainhash.Hash{}
	if err := ssz.Decode(reader, a.JustifiedBlockHash); err != nil {
		return err
	}

	return nil
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
	AggregateSig bls.Signature
}

// EncodeSSZ implements Encodable
func (a Attestation) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, a.Data); err != nil {
		return err
	}
	if err := ssz.Encode(writer, a.ParticipationBitfield); err != nil {
		return err
	}
	if err := ssz.Encode(writer, a.CustodyBitfield); err != nil {
		return err
	}
	if err := ssz.Encode(writer, a.AggregateSig); err != nil {
		return err
	}
	return nil
}

// EncodeSSZSize implements Encodable
func (a Attestation) EncodeSSZSize() (uint32, error) {
	var sizeOfdata, sizeOfparticipationBitfield, sizeOfcustodyBitfield, sizeOfaggregateSig uint32
	var err error
	if sizeOfdata, err = ssz.EncodeSize(a.Data); err != nil {
		return 0, err
	}
	if sizeOfparticipationBitfield, err = ssz.EncodeSize(a.ParticipationBitfield); err != nil {
		return 0, err
	}
	if sizeOfcustodyBitfield, err = ssz.EncodeSize(a.CustodyBitfield); err != nil {
		return 0, err
	}
	if sizeOfaggregateSig, err = ssz.EncodeSize(a.AggregateSig); err != nil {
		return 0, err
	}
	return sizeOfdata + sizeOfparticipationBitfield + sizeOfcustodyBitfield + sizeOfaggregateSig, nil
}

// DecodeSSZ implements Decodable
func (a Attestation) DecodeSSZ(reader io.Reader) error {
	a.Data = AttestationData{}
	if err := ssz.Decode(reader, a.Data); err != nil {
		return err
	}
	a.ParticipationBitfield = []uint8{}
	if err := ssz.Decode(reader, a.ParticipationBitfield); err != nil {
		return err
	}
	a.CustodyBitfield = []uint8{}
	if err := ssz.Decode(reader, a.CustodyBitfield); err != nil {
		return err
	}
	a.AggregateSig = bls.Signature{}
	if err := ssz.Decode(reader, a.AggregateSig); err != nil {
		return err
	}
	return nil
}

// Copy returns a copy of the attestation
func (a *Attestation) Copy() Attestation {
	sig := a.AggregateSig.Copy()
	return Attestation{
		Data:                  a.Data.Copy(),
		ParticipationBitfield: a.ParticipationBitfield[:],
		CustodyBitfield:       a.CustodyBitfield[:],
		AggregateSig:          *sig,
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
		AggregateSig:          bls.Signature{},
	}, nil
}

// ToProto gets the protobuf representation of the attestation
func (a *Attestation) ToProto() *pb.Attestation {
	att := &pb.Attestation{}
	att.Data = a.Data.ToProto()
	att.ParticipationBitfield = a.ParticipationBitfield[:]
	att.CustodyBitfield = a.CustodyBitfield[:]
	att.AggregateSig = a.AggregateSig.Serialize()
	return att
}

// PendingAttestation is an attestation waiting to be included.
type PendingAttestation struct {
	Data                  AttestationData
	ParticipationBitfield []byte
	CustodyBitfield       []byte
	SlotIncluded          uint64
}

// EncodeSSZ implements Encodable
func (pa PendingAttestation) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, pa.Data); err != nil {
		return err
	}
	if err := ssz.Encode(writer, pa.ParticipationBitfield); err != nil {
		return err
	}
	if err := ssz.Encode(writer, pa.CustodyBitfield); err != nil {
		return err
	}
	if err := ssz.Encode(writer, pa.SlotIncluded); err != nil {
		return err
	}

	return nil
}

// EncodeSSZSize implements Encodable
func (pa PendingAttestation) EncodeSSZSize() (uint32, error) {
	var sizeOfData, sizeOfParticipationBitfield, sizeOfCustodyBitfield, sizeOfSlotIncluded uint32
	var err error
	if sizeOfData, err = ssz.EncodeSize(pa.Data); err != nil {
		return 0, err
	}
	if sizeOfParticipationBitfield, err = ssz.EncodeSize(pa.ParticipationBitfield); err != nil {
		return 0, err
	}
	if sizeOfCustodyBitfield, err = ssz.EncodeSize(pa.CustodyBitfield); err != nil {
		return 0, err
	}
	if sizeOfSlotIncluded, err = ssz.EncodeSize(pa.SlotIncluded); err != nil {
		return 0, err
	}
	return sizeOfData + sizeOfParticipationBitfield + sizeOfCustodyBitfield + sizeOfSlotIncluded, nil
}

// DecodeSSZ implements Decodable
func (pa PendingAttestation) DecodeSSZ(reader io.Reader) error {
	pa.Data = AttestationData{}
	if err := ssz.Decode(reader, pa.Data); err != nil {
		return err
	}
	pa.ParticipationBitfield = []byte{}
	if err := ssz.Decode(reader, pa.ParticipationBitfield); err != nil {
		return err
	}
	pa.CustodyBitfield = []byte{}
	if err := ssz.Decode(reader, pa.CustodyBitfield); err != nil {
		return err
	}
	if err := ssz.Decode(reader, pa.SlotIncluded); err != nil {
		return err
	}

	return nil
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
