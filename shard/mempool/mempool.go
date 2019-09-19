package mempool

import (
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"sync"
)

// ValidationFunction validates a transaction for context-independent information (ensure signature is correct).
type ValidationFunction func([]byte) error

// ValidateTrue is a validation function that always evaluates to true.
var ValidateTrue ValidationFunction = func([]byte) error { return nil }

// PrioritizationFunction prioritizes a transaction based on fees or some other metric (could even be PoW).
type PrioritizationFunction func([]byte) float64

// PrioritizeEqual is a prioritization function that prioritizes all transaction equally.
var PrioritizeEqual PrioritizationFunction = func([]byte) float64 { return 1 }

// ShardMempool keeps track of a mempool for shards.
type ShardMempool struct {
	validationFunction     ValidationFunction
	prioritizationFunction PrioritizationFunction
	transactions           map[chainhash.Hash][]byte
	lock                   *sync.Mutex
}

// NewShardMempool creates a new shard mempool given a certain validation and prioritization function.
func NewShardMempool(v ValidationFunction, p PrioritizationFunction) *ShardMempool {
	return &ShardMempool{
		validationFunction:     v,
		prioritizationFunction: p,
		lock:                   new(sync.Mutex),
		transactions:           make(map[chainhash.Hash][]byte),
	}
}

// SubmitTransaction submits a transaction to the mempool.
func (sm *ShardMempool) SubmitTransaction(tx []byte) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	txHash := chainhash.HashH(tx)
	if _, found := sm.transactions[txHash]; found {
		// fail silently in the case we already have this transaction
		return nil
	}

	err := sm.validationFunction(tx)
	if err != nil {
		return err
	}

	sm.transactions[txHash] = tx

	return nil
}

// GetTransactions gets transactions from the mempool.
func (sm *ShardMempool) GetTransactions() [][]byte {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	transactions := make([][]byte, len(sm.transactions))

	i := 0
	for _, tx := range sm.transactions {
		transactions[i] = tx
		i++
	}

	return transactions
}

// RemoveTransactionsFromBlock removes transactions from the mempool from an accepted block.
func (sm *ShardMempool) RemoveTransactionsFromBlock(b *primitives.ShardBlock) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	for _, tx := range b.Body.Transactions {
		txBody := tx.TransactionData

		txHash := chainhash.HashH(txBody)

		delete(sm.transactions, txHash)
	}
}
