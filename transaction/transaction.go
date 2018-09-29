package transaction

import (
	"io"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"

	"github.com/phoreproject/synapse/serialization"
)

// Serializable represents an object that can be serialized
// or deserialized.
type Serializable interface {
	// Serialize takes an object an returns a representation of the object
	// in bytes.
	Serialize() []byte

	// Deserialize reads from a buffer and creates the object from a
	Deserialize(io.Reader) error
}

// GetHash returns the hash of a serializable object.
func GetHash(s Serializable) chainhash.Hash {
	serialized := s.Serialize()
	return chainhash.HashH(serialized)
}

// Transaction is a wrapper struct to provide functions for different
// transaction types.
type Transaction struct {
	Data      Serializable
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
	err = t.Data.Deserialize(r)
	if err != nil {
		return err
	}

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
func (t Transaction) Serialize() []byte {
	var transactionType byte
	switch t.Data.(type) {
	case TransferTransaction:
		transactionType = byte(typeTransfer)
	case RegisterTransaction:
		transactionType = byte(typeRegister)
	case SubmitAttestationTransaction:
		transactionType = byte(typeSubmitAttestation)
	case LoginTransaction:
		transactionType = byte(typeLogin)
	case LogoutTransaction:
		transactionType = byte(typeLogout)
	case FraudTransaction:
		transactionType = byte(typeFraud)
	}
	return append([]byte{transactionType}, t.Data.Serialize()...)
}
