package transaction

import (
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// RandaoChangeTransaction occurs when the randao
type RandaoChangeTransaction struct {
	ProposerIndex uint32
	NewRandao     chainhash.Hash
}

// Serialize serializes a login transaction to a 2d byte array
func (rct *RandaoChangeTransaction) Serialize() [][]byte {
	var fromBytes [4]byte
	binary.BigEndian.PutUint32(fromBytes[:], rct.ProposerIndex)
	return [][]byte{
		fromBytes[:],
		rct.NewRandao[:],
	}
}

// DeserializeRandaoChangeTransaction deserializes a 2d byte array into
// a randao change transaction.
func DeserializeRandaoChangeTransaction(b [][]byte) (*RandaoChangeTransaction, error) {
	if len(b) != 2 {
		return nil, fmt.Errorf("invalid login transaction")
	}

	if len(b[0]) != 4 {
		return nil, fmt.Errorf("invalid from validator index in login transaction")
	}

	if len(b[1]) > 32 {
		return nil, fmt.Errorf("invalid new Randao value")
	}

	proposerIndex := binary.BigEndian.Uint32(b[0])

	newRandao, err := chainhash.NewHash(b[1])
	if err != nil {
		return nil, err
	}

	return &RandaoChangeTransaction{ProposerIndex: proposerIndex, NewRandao: *newRandao}, nil
}
