package transaction

import (
	"github.com/phoreproject/synapse/serialization"
)

// RegisterTransaction registers a validator to be in the queue.
type RegisterTransaction struct {
	From serialization.Address
}
