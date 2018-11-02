package transaction

import (
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// RandaoRevealTransaction is used to update a RANDAO
type RandaoRevealTransaction struct {
	ValidatorIndex uint32
	Commitment     chainhash.Hash
}

// Serialize serializes a randao reveal transaction to a 2d byte array
func (rrt *RandaoRevealTransaction) Serialize() [][]byte {
	var fromBytes [4]byte
	binary.BigEndian.PutUint32(fromBytes[:], rrt.ValidatorIndex)
	return [][]byte{
		fromBytes[:],
		rrt.Commitment[:],
	}
}

// DeserializeRandaoRevealTransaction deserializes a 2d byte array into
// a randao reveal transaction.
func DeserializeRandaoRevealTransaction(b [][]byte) (*RandaoRevealTransaction, error) {
	if len(b) != 2 {
		return nil, fmt.Errorf("invalid regirandao revealster transaction")
	}

	if len(b[0]) != 4 {
		return nil, fmt.Errorf("invalid from address in randao reveal transaction")
	}

	if len(b[1]) != 32 {
		return nil, fmt.Errorf("invalid commitment in randao reveal transaction")
	}

	from := binary.BigEndian.Uint32(b[0])

	var commitment chainhash.Hash
	copy(commitment[:], b[1])

	return &RandaoRevealTransaction{ValidatorIndex: from, Commitment: commitment}, nil
}
