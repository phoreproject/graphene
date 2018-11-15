package transaction

import (
	pb "github.com/phoreproject/synapse/pb"
)

// Transaction is a wrapper struct to provide functions for different
// transaction types.
type Transaction struct {
	Data interface{}
}

const (
	typeRegister = iota
	typeLogin
	typeLogout
	typeCasperSlashing
	typeRandaoReveal
	typeRandaoChange
	typeInvalid = 9999
)

// DeserializeTransaction reads from a reader and deserializes a transaction into the
// specified type.
func DeserializeTransaction(transactionType uint32, transactionData [][]byte) (*Transaction, error) {
	t := &Transaction{}

	switch transactionType {
	case typeRegister:
		tx, err := DeserializeRegisterTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
	case typeLogin:
		tx, err := DeserializeLoginTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
	case typeLogout:
		tx, err := DeserializeLogoutTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
	case typeCasperSlashing:
		tx, err := DeserializeCasperSlashingTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
	case typeRandaoReveal:
		tx, err := DeserializeRandaoRevealTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
	case typeRandaoChange:
		tx, err := DeserializeRandaoChangeTransaction(transactionData)
		if err != nil {
			return nil, err
		}
		t.Data = tx
	}

	return t, nil
}

// Serialize serializes a transaction into binary.
func (t Transaction) Serialize() *pb.Special {
	var transactionType uint32
	var out [][]byte
	switch v := t.Data.(type) {
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
		out = v.Serialize()
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
		return &pb.Special{Type: typeInvalid, Data: [][]byte{}}
	}
	return &pb.Special{Type: transactionType, Data: out}
}
