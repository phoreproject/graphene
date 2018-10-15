package primitives

import (
	"bytes"
	"encoding/binary"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/transaction"
)

// BlockHeader represents a single beacon chain header.
type BlockHeader struct {
	SlotNumber            uint64
	RandaoReveal          chainhash.Hash
	AncestorHashes        []chainhash.Hash
	ActiveStateRoot       chainhash.Hash
	CrystallizedStateRoot chainhash.Hash
}

// Hash gets the hash of the block header
func (b BlockHeader) Hash() chainhash.Hash {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, b)
	return chainhash.HashH(buf.Bytes())
}

// Block represents a single beacon chain block.
type Block struct {
	BlockHeader
	Transactions []transaction.Transaction
}
