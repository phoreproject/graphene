package transaction

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/phoreproject/synapse/serialization"
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
	case typeCasperSlashing:
		t.Data = &CasperSlashingTransaction{}
	case typeRandaoReveal:
		t.Data = &RandaoRevealTransaction{}
	}

	tx, err := serialization.ReadByteArray(r)
	if err != nil {
		return err
	}
	r2 := bytes.NewBuffer(tx)
	binary.Read(r2, binary.BigEndian, &t.Data)
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
	case CasperSlashingTransaction:
		transactionType = byte(typeCasperSlashing)
		err = binary.Write(b, binary.BigEndian, t.Data.(CasperSlashingTransaction))
	case RandaoRevealTransaction:
		transactionType = byte(typeRandaoReveal)
		err = binary.Write(b, binary.BigEndian, t.Data.(RandaoRevealTransaction))
	}
	if err != nil {
		return []byte{}, err
	}
	return append([]byte{transactionType}, b.Bytes()...), nil
}
