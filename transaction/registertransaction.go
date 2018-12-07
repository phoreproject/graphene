package transaction

import (
	"fmt"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/serialization"
)

// RegisterTransaction registers a validator to be in the queue.
type RegisterTransaction struct {
	From      serialization.Address
	Signature bls.Signature
}

// Serialize serializes a register transaction to a 2d byte array
func (rt *RegisterTransaction) Serialize() [][]byte {
	return [][]byte{
		rt.From[:],
		rt.Signature.Serialize(),
	}
}

// DeserializeRegisterTransaction deserializes a 2d byte array into
// a register transaction.
func DeserializeRegisterTransaction(b [][]byte) (*RegisterTransaction, error) {
	if len(b) != 2 {
		return nil, fmt.Errorf("invalid register transaction")
	}

	if len(b[0]) != 20 {
		return nil, fmt.Errorf("invalid from address in register transaction")
	}

	fromAddr := serialization.Address{}

	copy(fromAddr[:], b[0])

	sig, err := bls.DeserializeSignature(b[1])
	if err != nil {
		return nil, err
	}

	return &RegisterTransaction{From: fromAddr, Signature: *sig}, nil
}
