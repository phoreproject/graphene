package primitives

import (
	"bytes"
	"encoding/binary"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/transaction"
)

// To create a block:
// - increase slot number
// - reveal randao commitment
// - update ancestor hashes
// - calculate state roots
// - aggregate specials + attestations (mempool?)

// Block represents a single beacon chain block.
type Block struct {
	SlotNumber            uint64
	RandaoReveal          chainhash.Hash
	AncestorHashes        []chainhash.Hash
	ActiveStateRoot       chainhash.Hash
	CrystallizedStateRoot chainhash.Hash
	Specials              []transaction.Transaction
	Attestations          []transaction.Attestation
}

// Hash gets the hash of the block header
func (b *Block) Hash() chainhash.Hash {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, b)
	return chainhash.HashH(buf.Bytes())
}
