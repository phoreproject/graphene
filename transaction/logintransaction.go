package transaction

import (
	"encoding/binary"
	"fmt"

	"github.com/phoreproject/synapse/bls"
)

// LoginTransaction queues a validator for login.
type LoginTransaction struct {
	From      uint32
	Signature bls.Signature
}

// Serialize serializes a login transaction to a 2d byte array
func (lt *LoginTransaction) Serialize() [][]byte {
	var fromBytes [4]byte
	binary.BigEndian.PutUint32(fromBytes[:], lt.From)
	return [][]byte{
		fromBytes[:],
		lt.Signature.Serialize(),
	}
}

// DeserializeLoginTransaction deserializes a 2d byte array into
// a login transaction.
func DeserializeLoginTransaction(b [][]byte) (*LoginTransaction, error) {
	if len(b) != 2 {
		return nil, fmt.Errorf("invalid login transaction")
	}

	if len(b[0]) != 4 {
		return nil, fmt.Errorf("invalid from validator index in login transaction")
	}

	from := binary.BigEndian.Uint32(b[0])

	sig, err := bls.DeserializeSignature(b[1])
	if err != nil {
		return nil, err
	}

	return &LoginTransaction{From: from, Signature: sig}, nil
}
