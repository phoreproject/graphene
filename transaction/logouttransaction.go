package transaction

import (
	"github.com/phoreproject/synapse/serialization"
)

// LogoutTransaction will queue a validator for logout.
type LogoutTransaction struct {
	From serialization.Address
}
