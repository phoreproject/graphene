package primitives

import (
	"io"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

// ProposalSignedData is a block proposal for a shard or beacon
// chain.
type ProposalSignedData struct {
	Slot      uint64
	Shard     uint64
	BlockHash chainhash.Hash
}

// EncodeSSZ implements Encodable
func (psd ProposalSignedData) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, psd.Slot); err != nil {
		return err
	}
	if err := ssz.Encode(writer, psd.Shard); err != nil {
		return err
	}
	if err := ssz.Encode(writer, psd.BlockHash); err != nil {
		return err
	}
	return nil
}

// EncodeSSZSize implements Encodable
func (psd ProposalSignedData) EncodeSSZSize() (uint32, error) {
	var sizeOfslot, sizeOfshard, sizeOfblockHash uint32
	var err error
	if sizeOfslot, err = ssz.EncodeSize(psd.Slot); err != nil {
		return 0, err
	}
	if sizeOfshard, err = ssz.EncodeSize(psd.Shard); err != nil {
		return 0, err
	}
	if sizeOfblockHash, err = ssz.EncodeSize(psd.BlockHash); err != nil {
		return 0, err
	}
	return sizeOfslot + sizeOfshard + sizeOfblockHash, nil
}

// DecodeSSZ implements Decodable
func (psd ProposalSignedData) DecodeSSZ(reader io.Reader) error {
	if err := ssz.Decode(reader, &psd.Slot); err != nil {
		return err
	}
	if err := ssz.Decode(reader, &psd.Shard); err != nil {
		return err
	}
	psd.BlockHash = chainhash.Hash{}
	if err := ssz.Decode(reader, &psd.BlockHash); err != nil {
		return err
	}
	return nil
}

// Copy returns a copy of the proposal signed data.
func (psd *ProposalSignedData) Copy() ProposalSignedData {
	return *psd
}

// ToProto gets the protobuf representation of a proposal signed data object
func (psd *ProposalSignedData) ToProto() pb.ProposalSignedData {
	return pb.ProposalSignedData{
		Slot:      psd.Slot,
		Shard:     psd.Shard,
		BlockHash: psd.BlockHash[:],
	}
}

// Hash gets the hash of the psd
func (psd *ProposalSignedData) Hash() chainhash.Hash {
	return chainhash.Hash{} // TODO: fixme
}
