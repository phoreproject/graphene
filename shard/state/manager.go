package state

import (
	"fmt"
	"sync"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/sirupsen/logrus"
)

type stateInfo struct {
	db     csmt.TreeDatabase
	state  State
	parent *stateInfo
	slot   uint64
	hash chainhash.Hash
}

// ShardStateManager keeps track of state by reverting and applying blocks.
type ShardStateManager struct {
	stateLock *sync.RWMutex
	// each of these TreeDatabase's is a TreeMemoryCache except for the finalized block which is the regular state DB.
	// When finalizing a block, we commit the hash of the finalized block and recurse through each parent until we reach
	// the previously justified block (which should be a real DB, not a cache).
	finalizedDB *stateInfo
	stateMap    map[chainhash.Hash]*stateInfo
	tipDB       *stateInfo

	shardInfo ShardInfo
}

// NewShardStateManager constructs a new shard state manager which keeps track of shard state.
func NewShardStateManager(stateDB csmt.TreeDatabase, stateSlot uint64, tipBlockHash chainhash.Hash, shardInfo ShardInfo) *ShardStateManager {
	stateHash, err := stateDB.Hash()
	if err != nil {
		panic(err)
	}

	tip := &stateInfo{
		db: stateDB,
		state: State{
			SmartContractRoot:   *stateHash,
			LastCrosslinkHash:   chainhash.Hash{},
			ProposerAssignments: []bls.PublicKey{},
		},
		hash: tipBlockHash,
		slot: stateSlot,
	}

	return &ShardStateManager{
		finalizedDB: tip,
		tipDB:       tip,
		shardInfo:   shardInfo,
		stateMap: map[chainhash.Hash]*stateInfo{
			tipBlockHash: tip,
		},
		stateLock: new(sync.RWMutex),
	}
}

// GetTip gets the state of the current tip.
func (sm *ShardStateManager) GetTip() (State, csmt.TreeDatabase) {
	sm.stateLock.Lock()
	defer sm.stateLock.Unlock()
	return sm.tipDB.state, sm.tipDB.db
}

// GetTipSlot gets the slot of the tip of the blockchain.
func (sm *ShardStateManager) GetTipSlot() uint64 {
	sm.stateLock.Lock()
	defer sm.stateLock.Unlock()

	return sm.tipDB.slot
}

func (sm *ShardStateManager) has(c chainhash.Hash) bool {
	_, found := sm.stateMap[c]
	return found
}

// Has checks if the ShardStateManager has a certain state for a block.
func (sm *ShardStateManager) Has(c chainhash.Hash) bool {
	sm.stateLock.RLock()
	defer sm.stateLock.RUnlock()
	return sm.has(c)
}

func executeBlockTransactions(a csmt.TreeTransactionAccess, b primitives.ShardBlock, si ShardInfo) error {
	for _, tx := range b.Body.Transactions {
		_, err := Transition(a, tx.TransactionData, si)
		if err != nil {
			return err
		}
	}
	return nil
}

// Add adds a block to the state map.
func (sm *ShardStateManager) Add(block *primitives.ShardBlock) (*chainhash.Hash, error) {
	sm.stateLock.Lock()
	defer sm.stateLock.Unlock()

	blockHash, _ := ssz.HashTreeRoot(block)
	if sm.has(blockHash) {
		// we already got this block
		return nil, nil
	}

	logrus.WithField("block hash", chainhash.Hash(blockHash)).WithField("shard", sm.shardInfo.ShardID).Debug("add block action")

	previousState, found := sm.stateMap[block.Header.PreviousBlockHash]
	if !found {
		return nil, fmt.Errorf("could not find parent block with hash: %s", block.Header.PreviousBlockHash)
	}

	newCache, err := csmt.NewTreeMemoryCache(previousState.db)
	if err != nil {
		return nil, err
	}

	newCacheTree := csmt.NewTree(newCache)
	err = newCacheTree.Update(func(access csmt.TreeTransactionAccess) error {
		return executeBlockTransactions(access, *block, sm.shardInfo)
	})
	if err != nil {
		return nil, err
	}

	sm.stateMap[blockHash] = &stateInfo{
		db:     newCache,
		parent: previousState,
		slot:   block.Header.Slot,
		hash: blockHash,
	}

	return newCache.Hash()
}

// Finalize removes unnecessary state from the state map.
func (sm *ShardStateManager) Finalize(finalizedHash chainhash.Hash, finalizedSlot uint64) error {
	sm.stateLock.Lock()
	defer sm.stateLock.Unlock()

	if sm.finalizedDB.hash.IsEqual(&finalizedHash) {
		return nil
	}

	logrus.WithField("block hash", finalizedHash).WithField("shard", sm.shardInfo.ShardID).Debug("finalize block action")

	// first, let's start at the tip and commit everything
	finalizeNode, found := sm.stateMap[finalizedHash]
	if !found {
		return fmt.Errorf("could not find block to finalize %s on slot %d %d", finalizedHash, finalizedSlot, sm.shardInfo.ShardID)
	}

	// start at the finalized node
	flushingNode := finalizeNode

	// while the flushing database is a cache
	memCache, isCache := flushingNode.db.(*csmt.TreeMemoryCache)
	for isCache {
		if err := memCache.Flush(); err != nil {
			return err
		}
		// go to previous block
		prevBlock := flushingNode.parent

		// if this is null, that means something went wrong because there is no underlying store.
		memCache, isCache = prevBlock.db.(*csmt.TreeMemoryCache)

		flushingNode = prevBlock
	}


	diskCache := flushingNode.db
	finalizeNode.db = diskCache

	// after we've flushed all of that, we should clean up any states with slots before that.
	for k, node := range sm.stateMap {
		if node.slot > finalizedSlot {
			continue
		}
		if k.IsEqual(&finalizedHash) {
			continue
		}
		if node.slot < finalizedSlot {
			delete(sm.stateMap, k)
		}
		if node.slot == finalizedSlot && !k.IsEqual(&finalizedHash) {
			delete(sm.stateMap, k)
		}
		if node.parent == finalizeNode {
			// if the parent is a tree cache, let's update the underlying store if needed
			childCache, isCache := node.db.(*csmt.TreeMemoryCache)
			if isCache {
				if err := childCache.UpdateUnderlying(diskCache); err != nil {
					return err
				}
			}
		}
	}

	sm.finalizedDB = sm.stateMap[finalizedHash]

	return nil
}

// SetTip sets the tip of the state.
func (sm *ShardStateManager) SetTip(newTipHash chainhash.Hash) error {
	sm.stateLock.Lock()
	defer sm.stateLock.Unlock()

	logrus.WithField("block hash", newTipHash).WithField("shard", sm.shardInfo.ShardID).Debug("new tip block action")

	newTip, found := sm.stateMap[newTipHash]
	if !found {
		return fmt.Errorf("couldn't find state root for tip: %s", newTipHash)
	}

	sm.tipDB = newTip

	return nil
}

// HasAny checks what blocks are in the state map.
func (sm *ShardStateManager) HasAny() map[chainhash.Hash]bool {
	sm.stateLock.RLock()
	defer sm.stateLock.RUnlock()

	hasBlock := make(map[chainhash.Hash]bool)

	for k := range sm.stateMap {
		hasBlock[k] = true
	}

	return hasBlock
}
