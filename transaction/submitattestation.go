package transaction

import (
	"io"

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
	JustifiedBlockHash  *chainhash.Hash
	ShardBlockHash      *chainhash.Hash
	ObliqueParentHashes []byte
}

func (sat SubmitAttestationTransaction) Deserialize(r io.Reader) error {

}

func (sat SubmitAttestationTransaction) Serialize() []byte {

}
