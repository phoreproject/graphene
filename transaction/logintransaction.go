package transaction

import (
	"github.com/phoreproject/synapse/serialization"
)

// LoginTransaction queues a validator for login.
type LoginTransaction struct {
	From serialization.Address
}
