package transaction

import (
	"io"

	"github.com/phoreproject/synapse/serialization"
)

// RegisterTransaction registers a validator to be in the queue.
type RegisterTransaction struct {
	From serialization.Address
}

// Deserialize reads a register transaction from bytes
func (rt RegisterTransaction) Deserialize(r io.Reader) error {
	from, err := serialization.ReadAddress(r)
	if err != nil {
		return err
	}
	rt.From = *from
	return nil
}

// Serialize serializes a register transaction into bytes.
func (rt RegisterTransaction) Serialize() []byte {
	return rt.From[:]
}
