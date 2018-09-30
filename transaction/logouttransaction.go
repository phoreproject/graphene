package transaction

import (
	"io"

	"github.com/phoreproject/synapse/serialization"
)

// LogoutTransaction will queue a validator for logout.
type LogoutTransaction struct {
	From serialization.Address
}

// Deserialize reads a logout transaction from reader.
func (lt LogoutTransaction) Deserialize(r io.Reader) error {
	from, err := serialization.ReadAddress(r)
	if err != nil {
		return err
	}
	lt.From = *from
	return nil
}

// Serialize serializes a logout transaction to bytes.
func (lt LogoutTransaction) Serialize() []byte {
	return lt.From[:]
}
