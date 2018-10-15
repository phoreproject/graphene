package transaction

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// SubmitAttestationTransaction submits a signed attestation to the
// beacon chain.
type SubmitAttestationTransaction struct {
	AggregateSignature []byte
	AttesterBitField   []byte
	Attestation
}

// Attestation is a signed attestation of a shard block.
type Attestation struct {
	Slot                uint64
	ShardID             uint64
	JustifiedSlot       uint64
	JustifiedBlockHash  chainhash.Hash
	ShardBlockHash      chainhash.Hash
	ObliqueParentHashes []chainhash.Hash
	AttesterBitField    []byte
	AggregateSignature  []byte
}

// AttestationSignedData is the part of the attestation that is signed.
type AttestationSignedData struct {
	Version        int64
	Slot           int64
	Shard          int64
	ParentHashes   [32]chainhash.Hash
	ShardBlockHash chainhash.Hash
	JustifiedSlot  int64
}
