package transaction

import (
	"github.com/phoreproject/synapse/bls"
)

// LogoutTransaction will queue a validator for logout.
type LogoutTransaction struct {
	From      uint32
	Signature bls.Signature
}
