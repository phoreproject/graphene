package transaction

import (
	"io"

	"github.com/phoreproject/synapse/serialization"
)

// LoginTransaction queues a validator for login.
type LoginTransaction struct {
	From serialization.Address
}

// Deserialize reads a login transaction from the reader.
func (lt LoginTransaction) Deserialize(r io.Reader) error {
	from, err := serialization.ReadAddress(r)
	if err != nil {
		return err
	}
	lt.From = *from
	return nil
}

func (lt LoginTransaction) Serialize() []byte {

}
