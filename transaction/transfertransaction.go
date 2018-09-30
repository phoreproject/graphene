package transaction

import (
	"encoding/binary"
	"io"

	"github.com/phoreproject/synapse/serialization"
)

// TransferTransaction represents a transaction for transferring money
// from one address to another.
type TransferTransaction struct {
	From   serialization.Address
	To     serialization.Address
	Amount uint64
}

// Deserialize reads from a reader into a TransferTransaction
func (tt TransferTransaction) Deserialize(r io.Reader) error {
	from, err := serialization.ReadAddress(r)
	if err != nil {
		return err
	}
	tt.From = *from

	to, err := serialization.ReadAddress(r)
	if err != nil {
		return err
	}
	tt.To = *to

	amount, err := serialization.ReadUint64(r)
	if err != nil {
		return err
	}
	tt.Amount = amount
	return nil
}

// Serialize serializes a transfer transaction to bytes.
func (tt TransferTransaction) Serialize() []byte {
	var amountBytes []byte
	binary.BigEndian.PutUint64(amountBytes, tt.Amount)
	return serialization.AppendAll(tt.From[:], tt.To[:], amountBytes)
}
