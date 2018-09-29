package transaction

import (
	"io"

	"github.com/phoreproject/synapse/serialization"
)

// RegisterTransaction registers a validator to be in the queue.
type RegisterTransaction struct {
	From serialization.Address
}

func (rt RegisterTransaction) Deserialize(r io.Reader) error {
	from, err := serialization.ReadAddress(r)
	if err != nil {
		return err
	}
	tt.From = *from
	return nil
}

func (rt RegisterTransaction) Serialize() []byte {

}
