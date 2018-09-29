package transaction

import (
	"io"

	"github.com/phoreproject/synapse/serialization"
)

// LogoutTransaction will queue a validator for logout.
type LogoutTransaction struct {
	From serialization.Address
}

func (lt LogoutTransaction) Deserialize(r io.Reader) error {

}

func (lt LogoutTransaction) Serialize() []byte {

}
