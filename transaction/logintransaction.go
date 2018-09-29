package transaction

import (
	"io"

	"github.com/phoreproject/synapse/serialization"
)

// LoginTransaction queues a validator for login.
type LoginTransaction struct {
	From serialization.Address
}

func (lt LoginTransaction) Deserialize(r io.Reader) error {

}

func (lt LoginTransaction) Serialize() []byte {

}
