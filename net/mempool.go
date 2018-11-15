package net

import (
	"github.com/gogo/protobuf/proto"
	logger "github.com/inconshreveable/log15"
	"github.com/phoreproject/synapse/blockchain"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/transaction"
)

// Mempool stores transactions to be included in a block
type Mempool struct {
	transactions []*transaction.Transaction
	attestations []*transaction.Attestation
	blockchain   *blockchain.Blockchain
}

// NewMempool creates a new mempool
func NewMempool(b *blockchain.Blockchain) Mempool {
	return Mempool{
		transactions: []*transaction.Transaction{},
		attestations: []*transaction.Attestation{},
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
func (m *Mempool) ProcessAttestation(a *transaction.Attestation) error {
	logger.Debug("received new attestation")
	lb, err := m.blockchain.LastBlock()
	if err != nil {
		return err
	}
	err = m.blockchain.ValidateAttestation(a, lb, m.blockchain.GetConfig())
	if err != nil {
		return err
	}
	m.attestations = append(m.attestations, a)
	return nil
}

// ProcessNewTransaction processes raw bytes received from the network
// and adds it to the mempool if needed.
func (m *Mempool) ProcessNewTransaction(transactionRaw *Message) error {
	tx := pb.Special{}

	err := proto.Unmarshal(transactionRaw.Data, &tx)
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
func (m *Mempool) ProcessNewAttestation(attestationRaw *Message) error {
	att := pb.Attestation{}

	err := proto.Unmarshal(attestationRaw.Data, &att)
	if err != nil {
		return err
	}

	a, err := transaction.NewAttestationFromProto(&att)
	if err != nil {
		return err
	}

	return m.ProcessAttestation(a)
}

// RegisterNetworkListeners registers listeners for mempool transactions
// and attestations.
func (m *Mempool) RegisterNetworkListeners(n *NetworkingService) error {
	err := n.RegisterHandler("tx", func(msg Message) error {
		return m.ProcessNewTransaction(&msg)
	})
	if err != nil {
		return err
	}
	return n.RegisterHandler("att", func(msg Message) error {
		return m.ProcessNewAttestation(&msg)
	})
}
