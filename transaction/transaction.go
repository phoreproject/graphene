package transaction

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/btcsuite/btcd/btcec"

	"github.com/phoreproject/synapse/serialization"
)

// Transaction is a wrapper struct to provide functions for different
// transaction types.
type Transaction struct {
	Data      interface{}
	Signed    bool
	Signature *btcec.Signature
}

const (
	typeTransfer = iota
	typeRegister
	typeSubmitAttestation
	typeLogin
	typeLogout
	typeFraud
)

// Deserialize reads from a reader and deserializes a transaction into the
// specified type.
func (t Transaction) Deserialize(r io.Reader) error {
	b, err := serialization.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	transactionType := uint8(b[0])
	switch transactionType {
	case typeTransfer:
		t.Data = &TransferTransaction{}
	case typeRegister:
		t.Data = &RegisterTransaction{}
	case typeLogin:
		t.Data = &LoginTransaction{}
	case typeLogout:
		t.Data = &LogoutTransaction{}
	case typeFraud:
		t.Data = &FraudTransaction{}
	}

	tx, err := serialization.ReadByteArray(r)
	if err != nil {
		return err
	}
	r2 := bytes.NewBuffer(tx)
	binary.Read(r2, binary.BigEndian, &t.Data)

	isSigned, err := serialization.ReadBool(r)
	if err != nil {
		return err
	}
	t.Signed = isSigned
	if isSigned {
		t.Signature = new(btcec.Signature)
		t.Signature.R, err = serialization.ReadBigInt(r)
		if err != nil {
			return err
		}
		t.Signature.S, err = serialization.ReadBigInt(r)
		if err != nil {
			return err
		}
	}
	return nil
}

// Serialize serializes a transaction into binary.
func (t Transaction) Serialize() ([]byte, error) {
	var transactionType byte
	b := new(bytes.Buffer)
	var err error
	switch t.Data.(type) {
	case TransferTransaction:
		transactionType = byte(typeTransfer)
		err = binary.Write(b, binary.BigEndian, t.Data.(TransferTransaction))
	case RegisterTransaction:
		transactionType = byte(typeRegister)
		err = binary.Write(b, binary.BigEndian, t.Data.(RegisterTransaction))
	case SubmitAttestationTransaction:
		transactionType = byte(typeSubmitAttestation)
		err = binary.Write(b, binary.BigEndian, t.Data.(SubmitAttestationTransaction))
	case LoginTransaction:
		transactionType = byte(typeLogin)
		err = binary.Write(b, binary.BigEndian, t.Data.(LoginTransaction))
	case LogoutTransaction:
		transactionType = byte(typeLogout)
		err = binary.Write(b, binary.BigEndian, t.Data.(LogoutTransaction))
	case FraudTransaction:
		transactionType = byte(typeFraud)
		err = binary.Write(b, binary.BigEndian, t.Data.(FraudTransaction))
	}
	if err != nil {
		return []byte{}, err
	}
	return append([]byte{transactionType}, b.Bytes()...), nil
}
