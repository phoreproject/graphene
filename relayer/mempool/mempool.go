package mempool

import (
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/execution"
	"sync"

	logger "github.com/sirupsen/logrus"
)

type shardMempoolItem struct {
	tx []byte
}

// ShardMempool keeps track of shard transactions
type ShardMempool struct {
	mempoolLock *sync.RWMutex
	shardInfo   execution.ShardInfo

	mempool           map[chainhash.Hash]*shardMempoolItem
	mempoolOrder      []*shardMempoolItem
	stateDB           csmt.TreeDatabase
}

// NewShardMempool constructs a new shard mempool.
func NewShardMempool(stateDB csmt.TreeDatabase, info execution.ShardInfo) *ShardMempool {
	return &ShardMempool{
		mempool:           make(map[chainhash.Hash]*shardMempoolItem),
		mempoolOrder:      nil,
		mempoolLock:       new(sync.RWMutex),
		stateDB:           stateDB,
		shardInfo:         info,
	}
}

func (s *ShardMempool) check(tx []byte) error {
	treeCache, err := csmt.NewTreeMemoryCache(s.stateDB)
	if err != nil {
		return err
	}
	tree := csmt.NewTree(treeCache)

	return tree.Update(func(treeTx csmt.TreeTransaction) error {
		_, err := execution.Transition(treeTx, tx, s.shardInfo)
		return err
	})
}

// Add adds a transaction to the mempool.
func (s *ShardMempool) Add(tx []byte) error {
	txHash := chainhash.HashH(tx)
	s.mempoolLock.Lock()
	defer s.mempoolLock.Unlock()
	if _, found := s.mempool[txHash]; found {
		logger.WithField("hash", txHash).Debug("transaction already exists")

		// don't do anything if it already exists
		return nil
	}

	if err := s.check(tx); err != nil {
		return err
	}

	poolItem := &shardMempoolItem{
		tx: tx,
	}

	s.mempool[txHash] = poolItem
	s.mempoolOrder = append(s.mempoolOrder, poolItem)

	return nil
}

// GetTransactions gets transactions to include.
func (s *ShardMempool) GetTransactions(maxBytes int) ([][]byte, *chainhash.Hash, error) {
	transactionsToInclude := make([][]byte, 0)
	s.mempoolLock.RLock()
	defer s.mempoolLock.RUnlock()

	totalBytes := 0

	packageTreeCache, err := csmt.NewTreeMemoryCache(s.stateDB)
	if err != nil {
		return nil, nil, err
	}

	// we have two layers of transactions here. The first layer, we want to rollback if something outright fails. This
	// could happen when we can't access part of the state for one reason or another.
	// The second layer we want to rollback if a transaction doesn't fit into the current execution state. For example,
	// a previous transaction may invalidate the next transaction.

	for _, tx := range s.mempoolOrder {
		if totalBytes + len(tx.tx) > maxBytes && maxBytes != -1 {
			continue
		}

		transactionTreeCache, err := csmt.NewTreeMemoryCache(packageTreeCache)
		if err != nil {
			return nil, nil, err
		}
		tree := csmt.NewTree(transactionTreeCache)

		err = tree.Update(func(treeTx csmt.TreeTransaction) error {
			// try to transition
			_, err := execution.Transition(treeTx, tx.tx, s.shardInfo)
			return err
		})
		if err != nil {
			continue
		}

		totalBytes += len(tx.tx)

		transactionsToInclude = append(transactionsToInclude, tx.tx)

		err = transactionTreeCache.Flush()
		if err != nil {
			return nil, nil, err
		}
	}

	endHash := csmt.EmptyTree

	err = packageTreeCache.View(func(tx csmt.TreeDatabaseTransaction) error {
		rootNode, err := tx.Root()
		if err != nil {
			return err
		}
		if rootNode != nil {
			endHash = rootNode.GetHash()
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return transactionsToInclude, &endHash, nil
}

// RemoveTransactionsFromBlock removes transactions from the mempool that were included in a block.
func (s *ShardMempool) RemoveTransactionsFromBlock(block *primitives.ShardBlock) {
	s.mempoolLock.Lock()
	defer s.mempoolLock.Unlock()

	for _, tx := range block.Body.Transactions {
		txHash := chainhash.HashH(tx.TransactionData)
		delete(s.mempool, txHash)
	}
}
