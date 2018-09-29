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

// Deserialize reads a fraudtransaction from the provided reader.
func (ft FraudTransaction) Deserialize(r io.Reader) error {
	from, err := serialization.ReadAddress(r)
	if err != nil {
		return err
	}
	ft.From = *from
	fraud, err := serialization.ReadByteArray(r)
	if err != nil {
		return err
	}

	ft.Fraud = fraud
	return nil
}

func (ft FraudTransaction) Serialize() []byte {

}
