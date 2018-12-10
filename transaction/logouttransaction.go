package transaction

import (
	"encoding/binary"
	"fmt"

	"github.com/phoreproject/synapse/bls"
)

// LogoutTransaction will queue a validator for logout.
type LogoutTransaction struct {
	From      uint32
	Signature bls.Signature
}

// Serialize serializes a logout transaction to a 2d byte array
func (lt *LogoutTransaction) Serialize() [][]byte {
	var fromBytes [4]byte
	binary.BigEndian.PutUint32(fromBytes[:], lt.From)
	return [][]byte{
		fromBytes[:],
		lt.Signature.Serialize(),
	}
}

// DeserializeLogoutTransaction deserializes a 2d byte array into
// a logout transaction.
func DeserializeLogoutTransaction(b [][]byte) (*LogoutTransaction, error) {
	if len(b) != 2 {
		return nil, fmt.Errorf("invalid logout transaction")
	}

	if len(b[0]) != 4 {
		return nil, fmt.Errorf("invalid from address in logout transaction")
	}

	from := binary.BigEndian.Uint32(b[0])

	sig, err := bls.DeserializeSignature(b[1])
	if err != nil {
		return nil, err
	}

	return &LogoutTransaction{From: from, Signature: *sig}, nil
}
