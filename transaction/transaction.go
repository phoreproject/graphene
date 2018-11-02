package transaction

import (
	"fmt"

	pb "github.com/phoreproject/synapse/pb"
)

// Transaction is a wrapper struct to provide functions for different
// transaction types.
type Transaction struct {
	Data interface{}
}

const (
	typeTransfer = iota
	typeRegister
	typeSubmitAttestation
	typeLogin
	typeLogout
	typeCasperSlashing
	typeRandaoReveal
)

// DeserializeTransaction reads from a reader and deserializes a transaction into the
// specified type.
func DeserializeTransaction(transactionType uint32, transactionData [][]byte) (*Transaction, error) {
	t := &Transaction{}

	switch transactionType {
	case typeTransfer:
		transferTransaction, err := DeserializeTransferTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = transferTransaction
	case typeRegister:
		transferTransaction, err := DeserializeRegisterTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = transferTransaction
	case typeLogin:
		transferTransaction, err := DeserializeLoginTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = transferTransaction
	case typeLogout:
		transferTransaction, err := DeserializeLogoutTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = transferTransaction
	case typeCasperSlashing:
		transferTransaction, err := DeserializeCasperSlashingTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = transferTransaction
	case typeRandaoReveal:
		transferTransaction, err := DeserializeRandaoRevealTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = transferTransaction
	}

	return t, nil
}

// Serialize serializes a transaction into binary.
func (t Transaction) Serialize() (*pb.Special, error) {
	var transactionType uint32
	var out [][]byte
	var err error
	switch v := t.Data.(type) {
	case TransferTransaction:
		transactionType = typeTransfer
		out = v.Serialize()
	case RegisterTransaction:
		transactionType = typeRegister
		out = v.Serialize()
	case LoginTransaction:
		transactionType = typeLogin
		out = v.Serialize()
	case LogoutTransaction:
		transactionType = typeLogout
		out = v.Serialize()
	case CasperSlashingTransaction:
		transactionType = typeCasperSlashing
		out, err = v.Serialize()
	case RandaoRevealTransaction:
		transactionType = typeRandaoReveal
		out = v.Serialize()
	default:
		return nil, fmt.Errorf("invalid transaction type")
	}

	if err != nil {
		return nil, err
	}

	return &pb.Special{Type: transactionType, Data: out}, nil
}
