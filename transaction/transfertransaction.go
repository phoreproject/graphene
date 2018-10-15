package transaction

import (
	"github.com/phoreproject/synapse/serialization"
)

// TransferTransaction represents a transaction for transferring money
// from one address to another.
type TransferTransaction struct {
	From   serialization.Address
	To     serialization.Address
	Amount uint64
}
