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
	typeRandaoChange
)

// DeserializeTransaction reads from a reader and deserializes a transaction into the
// specified type.
func DeserializeTransaction(transactionType uint32, transactionData [][]byte) (*Transaction, error) {
	t := &Transaction{}

	switch transactionType {
	case typeTransfer:
		tx, err := DeserializeTransferTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
		break
	case typeRegister:
		tx, err := DeserializeRegisterTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
		break
	case typeLogin:
		tx, err := DeserializeLoginTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
		break
	case typeLogout:
		tx, err := DeserializeLogoutTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
		break
	case typeCasperSlashing:
		tx, err := DeserializeCasperSlashingTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
		break
	case typeRandaoReveal:
		tx, err := DeserializeRandaoRevealTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
		break
	case typeRandaoChange:
		tx, err := DeserializeRandaoChangeTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
		break
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
		break
	case RegisterTransaction:
		transactionType = typeRegister
		out = v.Serialize()
		break
	case LoginTransaction:
		transactionType = typeLogin
		out = v.Serialize()
		break
	case LogoutTransaction:
		transactionType = typeLogout
		out = v.Serialize()
		break
	case CasperSlashingTransaction:
		transactionType = typeCasperSlashing
		out, err = v.Serialize()
		break
	case RandaoRevealTransaction:
		transactionType = typeRandaoReveal
		out = v.Serialize()
		break
	case RandaoChangeTransaction:
		transactionType = typeRandaoChange
		out = v.Serialize()
		break
	default:
		return nil, fmt.Errorf("invalid transaction type")
	}

	if err != nil {
		return nil, err
	}

	return &pb.Special{Type: transactionType, Data: out}, nil
}
