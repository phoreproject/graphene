package net

import (
	"github.com/gogo/protobuf/proto"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/transaction"
)

// Mempool stores transactions to be included in a block
type Mempool struct {
	transactions []*transaction.Transaction
	attestations []*transaction.Attestation
}

// NewMempool creates a new mempool
func NewMempool() Mempool {
	return Mempool{
		transactions: []*transaction.Transaction{},
		attestations: []*transaction.Attestation{},
	}
}

// ProcessTransaction validates a transaction and adds it to the mempool.
func (m *Mempool) ProcessTransaction(t *transaction.Transaction) {
	// TODO: mempool validation
	m.transactions = append(m.transactions, t)
}

// ProcessAttestation validates an attestation and adds it to the
// mempool.
func (m *Mempool) ProcessAttestation(a *transaction.Attestation) {
	m.attestations = append(m.attestations, a)
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

	m.ProcessTransaction(t)
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

	m.ProcessAttestation(a)
	return nil
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
