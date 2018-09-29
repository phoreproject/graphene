package transaction

import (
	"io"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"

	"github.com/phoreproject/phore2/serialization"
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

// TransferTransaction represents a transaction for transferring money
// from one address to another.
type TransferTransaction struct {
	From   serialization.Address
	To     serialization.Address
	Amount uint64
}

// RegisterTransaction registers a validator to be in the queue.
type RegisterTransaction struct {
	From serialization.Address
}

// SubmitAttestationTransaction submits a signed attestation to the
// beacon chain.
type SubmitAttestationTransaction struct {
	AggregateSignature []byte
	AttesterBitField   []byte
	Attestation
}

// Attestation is a signed attestation of a shard block.
type Attestation struct {
	Slot                uint64
	ShardID             uint64
	JustifiedSlot       uint64
	JustifiedBlockHash  *chainhash.Hash
	ShardBlockHash      *chainhash.Hash
	ObliqueParentHashes []byte
}

// LoginTransaction queues a validator for login.
type LoginTransaction struct {
	From serialization.Address
}

// LogoutTransaction will queue a validator for logout.
type LogoutTransaction struct {
	From serialization.Address
}

// FraudTransaction is used for reporting fraud in an attestation.
type FraudTransaction struct {
	From  serialization.Address
	Fraud []byte
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

// Deserialize reads from a reader into a TransferTransaction
func (tt TransferTransaction) Deserialize(r io.Reader) error {
	from, err := serialization.ReadAddress(r)
	if err != nil {
		return err
	}
	tt.From = *from

	to, err := serialization.ReadAddress(r)
	if err != nil {
		return err
	}
	tt.To = *to

	amount, err := serialization.ReadUint64(r)
	if err != nil {
		return err
	}
	tt.Amount = amount
	return nil
}

func (tt TransferTransaction) Serialize() []byte {

}

func (rt RegisterTransaction) Deserialize(r io.Reader) error {

}

func (rt RegisterTransaction) Serialize() []byte {

}

func (sat SubmitAttestationTransaction) Deserialize(r io.Reader) error {

}

func (sat SubmitAttestationTransaction) Serialize() []byte {

}

func (lt LoginTransaction) Deserialize(r io.Reader) error {

}

func (lt LoginTransaction) Serialize() []byte {

}

func (lt LogoutTransaction) Deserialize(r io.Reader) error {

}

func (lt LogoutTransaction) Serialize() []byte {

}

func (ft FraudTransaction) Deserialize(r io.Reader) error {

}

func (ft FraudTransaction) Serialize() []byte {

}
