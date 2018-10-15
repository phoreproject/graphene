package transaction

import (
	"github.com/phoreproject/synapse/serialization"
)

// FraudTransaction is used for reporting fraud in an attestation.
type FraudTransaction struct {
	From  serialization.Address
	Fraud []byte
}
