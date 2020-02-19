package mempool

import (
	"sync"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/state"
	"github.com/sirupsen/logrus"
)

type shardMempoolItem struct {
	tx []byte
}

// ShardMempool keeps track of shard transactions
type ShardMempool struct {
	mempoolLock *sync.RWMutex
	shardInfo   state.ShardInfo

	mempool      map[chainhash.Hash]*shardMempoolItem
	mempoolOrder []*shardMempoolItem

	stateManager *state.ShardStateManager
}

// NewShardMempool constructs a new shard mempool. This needs to keep track of fork choice and a state tree. Changes
// should be written to disk once finalized.
func NewShardMempool(stateDB csmt.TreeDatabase, stateSlot uint64, tipBlockHash chainhash.Hash, info state.ShardInfo) *ShardMempool {
	return &ShardMempool{
		mempool:      make(map[chainhash.Hash]*shardMempoolItem),
		mempoolOrder: nil,
		mempoolLock:  new(sync.RWMutex),
		stateManager: state.NewShardStateManager(stateDB, stateSlot, tipBlockHash, info),
		shardInfo: info,
	}
}

// GetTipState gets the state at the current tip.
func (s *ShardMempool) GetTipState() csmt.TreeDatabase {
	return s.stateManager.GetTip()
}

func (s *ShardMempool) check(tx []byte) error {
	treeCache, err := csmt.NewTreeMemoryCache(s.GetTipState())
	if err != nil {
		return err
	}
	tree := csmt.NewTree(treeCache)

	return tree.Update(func(treeTx csmt.TreeTransactionAccess) error {
		_, err := state.Transition(treeTx, tx, s.shardInfo)
		return err
	})
}

// Add adds a transaction to the mempool.
func (s *ShardMempool) Add(tx []byte) error {
	txHash := chainhash.HashH(tx)
	s.mempoolLock.Lock()
	defer s.mempoolLock.Unlock()

	logrus.WithField("hash", txHash).Info("adding new transaction to mempool")

	if _, found := s.mempool[txHash]; found {
		logrus.WithField("hash", txHash).Debug("transaction already exists")

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

// AcceptAction updates the current state with all of the
func (s *ShardMempool) AcceptAction(action *pb.ShardChainAction) error {
	// We have 3 types of actions:
	// 1. AddBlockAction - adds a block and executes it, adding the result to the state map
	// 2. FinalizeBlockAction - finalizes a block, deletes all states before it, and commits changes to disk
	// 3. UpdateTip - updates the current tip to a certain block hash

	// add block adds a block to the chain
	if action.AddBlockAction != nil {
		blockToAdd, err := primitives.ShardBlockFromProto(action.AddBlockAction.Block)
		if err != nil {
			return err
		}

		if _, err := s.stateManager.Add(blockToAdd); err != nil {
			return err
		}

		s.RemoveTransactionsFromBlock(blockToAdd)
	} else if action.FinalizeBlockAction != nil {
		toFinalizeHash, err := chainhash.NewHash(action.FinalizeBlockAction.Hash)
		if err != nil {
			return err
		}

		if err := s.stateManager.Finalize(*toFinalizeHash, action.FinalizeBlockAction.Slot); err != nil {
			return err
		}
	} else if action.UpdateTip != nil {
		newTipHash, err := chainhash.NewHash(action.UpdateTip.Hash)
		if err != nil {
			return err
		}

		if err := s.stateManager.SetTip(*newTipHash); err != nil {
			return err
		}
	}

	return nil
}

// GetCurrentRoot gets the current root of the mempool.
func (s *ShardMempool) GetCurrentRoot() (*chainhash.Hash, error) {
	return s.GetTipState().Hash()
}

// GetTransactions gets transactions to include.
func (s *ShardMempool) GetTransactions(maxBytes int) (*primitives.TransactionPackage, uint64, error) {
	transactionsToInclude := make([]primitives.ShardTransaction, 0)
	s.mempoolLock.RLock()
	defer s.mempoolLock.RUnlock()

	totalBytes := 0

	startRootHash, err := s.GetTipState().Hash()
	if err != nil {
		return nil, 0, err
	}

	startSlot := s.stateManager.GetTipSlot()

	var startRoot [32]byte
	copy(startRoot[:], startRootHash[:])

	packageTreeCache, err := csmt.NewTreeMemoryCache(s.GetTipState())
	if err != nil {
		return nil, 0, err
	}

	updates := make([]primitives.UpdateWitness, 0)
	verifications := make([]primitives.VerificationWitness, 0)

	// we have two layers of transactions here. The first layer, we want to rollback if something outright fails. This
	// could happen when we can't access part of the state for one reason or another.
	// The second layer we want to rollback if a transaction doesn't fit into the current execution state. For example,
	// a previous transaction may invalidate the next transaction.
	for _, tx := range s.mempoolOrder {
		if totalBytes+len(tx.tx) > maxBytes && maxBytes != -1 {
			continue
		}

		transactionTreeCache, err := csmt.NewTreeMemoryCache(packageTreeCache)
		if err != nil {
			return nil, 0, err
		}
		tree := csmt.NewTree(transactionTreeCache)

		trackingTree, err := state.NewTrackingState(tree)
		if err != nil {
			return nil, 0, err
		}

		err = trackingTree.Update(func(treeTx csmt.TreeTransactionAccess) error {
			// try to transition
			_, err := state.Transition(treeTx, tx.tx, s.shardInfo)
			return err
		})
		if err != nil {
			continue
		}

		totalBytes += len(tx.tx)

		transactionsToInclude = append(transactionsToInclude, primitives.ShardTransaction{
			TransactionData: tx.tx,
		})

		err = transactionTreeCache.Flush()
		if err != nil {
			return nil, 0, err
		}

		_, txVerifications, txUpdates := trackingTree.GetWitnesses()

		updates = append(updates, txUpdates...)
		verifications = append(verifications, txVerifications...)
	}

	endHash := primitives.EmptyTree

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
		return nil, 0, err
	}

	txPackage := &primitives.TransactionPackage{
		StartRoot:     startRoot,
		EndRoot:       endHash,
		Updates:       updates,
		Verifications: verifications,
		Transactions:  transactionsToInclude,
	}

	return txPackage, startSlot, nil
}

// RemoveTransactionsFromBlock removes transactions from the mempool that were included in a block.
func (s *ShardMempool) RemoveTransactionsFromBlock(block *primitives.ShardBlock) {
	s.mempoolLock.Lock()
	defer s.mempoolLock.Unlock()

	newOrder := make([]*shardMempoolItem, 0, len(s.mempoolOrder))
	for _, tx := range block.Body.Transactions {
		txHash := chainhash.HashH(tx.TransactionData)
		delete(s.mempool, txHash)
	}

	for _, tx := range s.mempoolOrder {
		txHash := chainhash.HashH(tx.tx)
		if _, found := s.mempool[txHash]; found {
			newOrder = append(newOrder, tx)
		}
	}

	s.mempoolOrder = newOrder
}
