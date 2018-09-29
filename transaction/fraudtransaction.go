package transaction

import (
	"io"

	"github.com/phoreproject/synapse/serialization"
)

// FraudTransaction is used for reporting fraud in an attestation.
type FraudTransaction struct {
	From  serialization.Address
	Fraud []byte
}

func (ft FraudTransaction) Deserialize(r io.Reader) error {

}

func (ft FraudTransaction) Serialize() []byte {

}
