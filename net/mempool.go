package net

import (
	"github.com/gogo/protobuf/proto"
	"github.com/phoreproject/synapse/blockchain"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/transaction"
	logger "github.com/sirupsen/logrus"
)

// Mempool stores transactions to be included in a block
type Mempool struct {
	transactions []*transaction.Transaction
	attestations []*transaction.AttestationRecord
	blockchain   *blockchain.Blockchain
}

// NewMempool creates a new mempool
func NewMempool(b *blockchain.Blockchain) Mempool {
	return Mempool{
		transactions: []*transaction.Transaction{},
		attestations: []*transaction.AttestationRecord{},
		blockchain:   b,
	}
}

// ProcessTransaction validates a transaction and adds it to the mempool.
func (m *Mempool) ProcessTransaction(t *transaction.Transaction) error {
	logger.Debug("received new transaction")

	// TODO: mempool validation
	m.transactions = append(m.transactions, t)

	return nil
}

// ProcessAttestation validates an attestation and adds it to the
// mempool.
func (m *Mempool) ProcessAttestation(a *transaction.AttestationRecord) error {
	logger.Debug("received new attestation")
	lb, err := m.blockchain.LastBlock()
	if err != nil {
		return err
	}
	err = m.blockchain.ValidateAttestationRecord(a, lb, m.blockchain.GetConfig())
	if err != nil {
		return err
	}
	m.attestations = append(m.attestations, a)
	return nil
}

// ProcessNewTransaction processes raw bytes received from the network
// and adds it to the mempool if needed.
func (m *Mempool) ProcessNewTransaction(transactionRaw []byte) error {
	tx := pb.Special{}

	err := proto.Unmarshal(transactionRaw, &tx)
	if err != nil {
		return err
	}

	t, err := transaction.DeserializeTransaction(tx.Type, tx.Data)
	if err != nil {
		return err
	}

	err = m.ProcessTransaction(t)
	if err != nil {
		return err
	}
	return nil
}

// ProcessNewAttestation processes raw bytes received from the network
// and adds it to the mempool if needed.
func (m *Mempool) ProcessNewAttestation(attestationRaw []byte) error {
	att := pb.AttestationRecord{}

	err := proto.Unmarshal(attestationRaw, &att)
	if err != nil {
		return err
	}

	a, err := transaction.NewAttestationRecordFromProto(&att)
	if err != nil {
		return err
	}

	return m.ProcessAttestation(a)
}

// RegisterNetworkListeners registers listeners for mempool transactions
// and attestations.
func (m *Mempool) RegisterNetworkListeners(n *NetworkingService) error {
	_, err := n.RegisterHandler("tx", func(msg []byte) error {
		return m.ProcessNewTransaction(msg)
	})
	if err != nil {
		return err
	}
	_, err = n.RegisterHandler("att", func(msg []byte) error {
		return m.ProcessNewAttestation(msg)
	})
	return err
}
