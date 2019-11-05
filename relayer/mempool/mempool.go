package mempool

import (
	"fmt"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/execution"
	"github.com/phoreproject/synapse/shard/state"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/sirupsen/logrus"
	"sync"
	)

type shardMempoolItem struct {
	tx []byte
}

type stateInfo struct {
	db csmt.TreeDatabase
	parent *stateInfo
	slot uint64
}

// ShardMempool keeps track of shard transactions
type ShardMempool struct {
	mempoolLock *sync.RWMutex
	shardInfo   execution.ShardInfo

	mempool      map[chainhash.Hash]*shardMempoolItem
	mempoolOrder []*shardMempoolItem

	stateLock *sync.RWMutex
	// each of these TreeDatabase's is a TreeMemoryCache except for the finalized block which is the regular state DB.
	// When finalizing a block, we commit the hash of the finalized block and recurse through each parent until we reach
	// the previously justified block (which should be a real DB, not a cache).
	finalizedDB   *stateInfo
	stateMap map[chainhash.Hash]*stateInfo
	tipDB *stateInfo
}

// NewShardMempool constructs a new shard mempool. This needs to keep track of fork choice and a state tree. Changes
// should be written to disk once finalized.
func NewShardMempool(stateDB csmt.TreeDatabase, stateSlot uint64, tipBlockHash chainhash.Hash, info execution.ShardInfo) *ShardMempool {
	tip := &stateInfo{
		db: stateDB,
		slot: stateSlot,
	}
	return &ShardMempool{
		mempool:      make(map[chainhash.Hash]*shardMempoolItem),
		mempoolOrder: nil,
		mempoolLock:  new(sync.RWMutex),
		finalizedDB: tip,
		tipDB: tip,
		stateMap: map[chainhash.Hash]*stateInfo{
			tipBlockHash: tip,
		},
		stateLock: new(sync.RWMutex),
		shardInfo:    info,
	}
}

func (s *ShardMempool) getTipState() csmt.TreeDatabase {
	s.stateLock.RLock()
	defer s.stateLock.RUnlock()
	return s.tipDB.db
}

func (s *ShardMempool) check(tx []byte) error {
	treeCache, err := csmt.NewTreeMemoryCache(s.getTipState())
	if err != nil {
		return err
	}
	tree := csmt.NewTree(treeCache)

	return tree.Update(func(treeTx csmt.TreeTransactionAccess) error {
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

func executeBlockTransactions(a csmt.TreeTransactionAccess, b primitives.ShardBlock, si execution.ShardInfo) error {
	for _, tx := range b.Body.Transactions {
		_, err := execution.Transition(a, tx.TransactionData, si)
		if err != nil {
			return err
		}
	}
	return nil
}

// AcceptBlock updates the current state with all of the
func (s *ShardMempool) AcceptAction(action *pb.ShardChainAction) error {
	// We have 3 types of actions:
	// 1. AddBlockAction - adds a block and executes it, adding the result to the state map
	// 2. FinalizeBlockAction - finalizes a block, deletes all states before it, and commits changes to disk
	// 3. UpdateTip - updates the current tip to a certain block hash

	// add block adds a block to the chain
	if action.AddBlockAction != nil {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()

		blockToAdd, err := primitives.ShardBlockFromProto(action.AddBlockAction.Block)
		if err != nil {
			return err
		}

		blockHash, _ := ssz.HashTreeRoot(blockToAdd)
		logrus.WithField("block hash", chainhash.Hash(blockHash)).Info("add block action")

		previousTree, found := s.stateMap[blockToAdd.Header.PreviousBlockHash]
		if !found {
			return fmt.Errorf("could not find parent block with hash: %s", blockToAdd.Header.PreviousBlockHash)
		}

		newCache, err := csmt.NewTreeMemoryCache(previousTree.db)
		if err != nil {
			return err
		}

		newCacheTree := csmt.NewTree(newCache)
		err = newCacheTree.Update(func(access csmt.TreeTransactionAccess) error {
			return executeBlockTransactions(access, *blockToAdd, s.shardInfo)
		})
		if err != nil {
			return err
		}

		s.stateMap[blockHash] = &stateInfo{
			db: newCache,
			parent: previousTree,
			slot: blockToAdd.Header.Slot,
		}
	} else if action.FinalizeBlockAction != nil {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()

		toFinalizeHash, err := chainhash.NewHash(action.FinalizeBlockAction.Hash)
		if err != nil {
			return err
		}

		logrus.WithField("block hash", toFinalizeHash).Info("finalize block action")

		// first, let's start at the tip and commit everything
		finalizeNode, found := s.stateMap[*toFinalizeHash]
		if !found {
			return fmt.Errorf("could not find block to finalize %s", toFinalizeHash)
		}
		memCache, isCache := finalizeNode.db.(*csmt.TreeMemoryCache)
		for isCache {
			if err := memCache.Flush(); err != nil {
				return err
			}
			prevBlock := finalizeNode.parent
			// if this is null, that means something went wrong because there is no underlying store.
			memCache, isCache = prevBlock.db.(*csmt.TreeMemoryCache)
		}

		// after we've flushed all of that, we should clean up any states with slots before that.
		for k, node := range s.stateMap {
			if node.slot > action.FinalizeBlockAction.Slot {
				continue
			}
			if node.slot < action.FinalizeBlockAction.Slot {
				delete(s.stateMap, k)
			}
			if node.slot == action.FinalizeBlockAction.Slot && !k.IsEqual(toFinalizeHash) {
				delete(s.stateMap, k)
			}
		}

		s.finalizedDB = s.stateMap[*toFinalizeHash]
	} else if action.UpdateTip != nil {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()

		newTipHash, err := chainhash.NewHash(action.UpdateTip.Hash)
		if err != nil {
			return err
		}

		logrus.WithField("block hash", newTipHash).Info("new tip block action")

		newTip, found := s.stateMap[*newTipHash]
		if !found {
			return fmt.Errorf("couldn't find state root for tip: %s", newTipHash)
		}

		s.tipDB = newTip
	}

	return nil
}

// GetCurrentRoot gets the current root of the mempool.
func (s *ShardMempool) GetCurrentRoot() (*chainhash.Hash, error) {
	return s.getTipState().Hash()
}

// GetTransactions gets transactions to include.
func (s *ShardMempool) GetTransactions(maxBytes int) (*primitives.TransactionPackage, error) {
	transactionsToInclude := make([]primitives.ShardTransaction, 0)
	s.mempoolLock.RLock()
	defer s.mempoolLock.RUnlock()

	totalBytes := 0

	startRoot, err := s.getTipState().Hash()
	if err != nil {
		return nil, err
	}

	packageTreeCache, err := csmt.NewTreeMemoryCache(s.getTipState())
	if err != nil {
		return nil, err
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
			return nil, err
		}
		tree := csmt.NewTree(transactionTreeCache)

		trackingTree, err := state.NewTrackingState(tree)
		if err != nil {
			return nil, err
		}

		err = trackingTree.Update(func(treeTx csmt.TreeTransactionAccess) error {
			// try to transition
			_, err := execution.Transition(treeTx, tx.tx, s.shardInfo)
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
			return nil, err
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
		return nil, err
	}

	if err := packageTreeCache.Flush(); err != nil {
		return nil, err
	}

	txPackage := &primitives.TransactionPackage{
		StartRoot:     *startRoot,
		EndRoot:       endHash,
		Updates:       updates,
		Verifications: verifications,
		Transactions:  transactionsToInclude,
	}

	return txPackage, nil
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
