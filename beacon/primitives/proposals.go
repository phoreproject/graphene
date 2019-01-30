package primitives

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/pb"
)

// ProposalSignedData is a block proposal for a shard or beacon
// chain.
type ProposalSignedData struct {
	Slot      uint64
	Shard     uint64
	BlockHash chainhash.Hash
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
