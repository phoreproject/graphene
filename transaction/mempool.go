package transaction

// Mempool stores transactions to be included in a block
type Mempool struct {
	transactions []*Transaction
	attestations []*Attestation
}

// NewMempool creates a new mempool
func NewMempool() Mempool {
	return Mempool{
		transactions: []*Transaction{},
		attestations: []*Attestation{},
	}
}

// ProcessTransaction validates a transaction and adds it to the mempool.
func (m *Mempool) ProcessTransaction(t *Transaction) {
	// TODO: mempool validation
	m.transactions = append(m.transactions, t)
}

// ProcessAttestation validates an attestation and adds it to the
// mempool.
func (m *Mempool) ProcessAttestation(a *Attestation) {
	m.attestations = append(m.attestations, a)
}
