package transaction

import (
	"encoding/binary"
	"fmt"

	"github.com/phoreproject/synapse/serialization"
)

// TransferTransaction represents a transaction for transferring money
// from one address to another.
type TransferTransaction struct {
	From   serialization.Address
	To     serialization.Address
	Amount uint64
}

// Serialize gets the binary representation of a transfer
// transaction.
func (tt *TransferTransaction) Serialize() [][]byte {
	var amountBytes [8]byte
	binary.BigEndian.PutUint64(amountBytes[:], tt.Amount)
	return [][]byte{
		tt.From[:],
		tt.To[:],
		amountBytes[:],
	}
}

// DeserializeTransferTransaction deserializes a transfer transaction
// from the binary representation.
func DeserializeTransferTransaction(b [][]byte) (*TransferTransaction, error) {
	if len(b) != 3 {
		return nil, fmt.Errorf("invalid transfer transaction")
	}

	addrFrom := serialization.Address{}
	addrTo := serialization.Address{}

	if len(b[0]) != 20 || len(b[1]) != 20 {
		return nil, fmt.Errorf("invalid address in transfer transaction")
	}

	copy(addrFrom[:], b[0])
	copy(addrTo[:], b[1])

	amount := binary.BigEndian.Uint64(b[2])
	return &TransferTransaction{From: addrFrom, To: addrTo, Amount: amount}, nil
}
