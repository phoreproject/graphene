package transaction

import (
	"bytes"
	"fmt"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/serialization"
)

// CasperSlashingTransaction is used for reporting fraud in an attestation.
type CasperSlashingTransaction struct {
	SourceValidators              []uint32
	SourceDataSigned              []byte
	SourceAggregateSignature      bls.Signature
	DestinationValidators         []uint32
	DestinationDataSigned         []byte
	DestinationAggregateSignature bls.Signature
}

// Serialize gets a binary representation of the transaction.
func (c *CasperSlashingTransaction) Serialize() [][]byte {
	return [][]byte{
		serialization.WriteUint32Array(c.SourceValidators),
		c.SourceDataSigned,
		c.SourceAggregateSignature.Serialize(),
		serialization.WriteUint32Array(c.DestinationValidators),
		c.DestinationDataSigned,
		c.DestinationAggregateSignature.Serialize(),
	}
}

// DeserializeCasperSlashingTransaction deserializes a binary casper
// slashing transaction
func DeserializeCasperSlashingTransaction(b [][]byte) (*CasperSlashingTransaction, error) {
	if len(b) != 6 {
		return nil, fmt.Errorf("invalid slashing transaction")
	}

	svBuf := bytes.NewBuffer(b[0])
	sv, err := serialization.ReadUint32Array(svBuf)
	if err != nil {
		return nil, err
	}

	sas, err := bls.DeserializeSignature(b[2])
	if err != nil {
		return nil, err
	}

	dvBuf := bytes.NewBuffer(b[3])
	dv, err := serialization.ReadUint32Array(dvBuf)
	if err != nil {
		return nil, err
	}

	das, err := bls.DeserializeSignature(b[5])
	if err != nil {
		return nil, err
	}

	return &CasperSlashingTransaction{
		SourceValidators:              sv,
		SourceDataSigned:              b[1],
		SourceAggregateSignature:      *sas,
		DestinationValidators:         dv,
		DestinationDataSigned:         b[2],
		DestinationAggregateSignature: *das,
	}, nil
}
