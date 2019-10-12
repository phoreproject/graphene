package csmt

import (
	"errors"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"sync"
)

// TreeMemoryCache is a cache that allows
type TreeMemoryCache struct {
	underlyingStore TreeDatabase
	dirty           map[chainhash.Hash]Node
	dirtyKV         map[chainhash.Hash]chainhash.Hash
	toRemove        map[chainhash.Hash]struct{}
	root chainhash.Hash

	lock *sync.RWMutex
}

// Hash gets the hash of the current tree.
func (t *TreeMemoryCache) Hash() (*chainhash.Hash, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return &t.root, nil
}

// NewTreeMemoryCache constructs a tree memory cache that can be committed based on an underlying tree (probably on disk).
func NewTreeMemoryCache(underlyingDatabase TreeDatabase) (*TreeMemoryCache, error) {
	var preRoot chainhash.Hash

	err := underlyingDatabase.View(func(tx TreeDatabaseTransaction) error {
		root, err := tx.Root()
		if err != nil {
			return err
		}

		if root == nil {
			preRoot = primitives.EmptyTree
			return nil
		}

		preRoot = root.GetHash()

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &TreeMemoryCache{
		underlyingStore: underlyingDatabase,
		root: preRoot,
		dirty: make(map[chainhash.Hash]Node),
		dirtyKV: make(map[chainhash.Hash]chainhash.Hash),
		toRemove: make(map[chainhash.Hash]struct{}),
		lock: new(sync.RWMutex),
	}, nil
}

// Update creates an update transaction for the cache.
func (t *TreeMemoryCache) Update(cb func(TreeDatabaseTransaction) error) error {
	return t.underlyingStore.View(func(underlyingTx TreeDatabaseTransaction) error {
		t.lock.Lock()
		defer t.lock.Unlock()

		// we copy and set so that we can easily revert if needed (if the transaction errors)
		dirtyCopy := make(map[chainhash.Hash]Node)
		dirtyKVCopy := make(map[chainhash.Hash]chainhash.Hash)
		toRemoveCopy := make(map[chainhash.Hash]struct{})

		for k, v := range t.dirty {
			dirtyCopy[k] = v
		}

		for k, v := range t.dirtyKV {
			dirtyKVCopy[k] = v
		}

		for k := range t.toRemove {
			toRemoveCopy[k] = struct{}{}
		}

		tx := &TreeMemoryCacheTransaction{
			underlyingTransaction: underlyingTx,
			root:                  t.root,
			dirty:                 dirtyCopy,
			dirtyKV:               dirtyKVCopy,
			toRemove:              toRemoveCopy,
			update:                true,
		}

		err := cb(tx)
		if err != nil {
			return err
		}

		t.root = tx.root
		t.dirty = dirtyCopy
		t.dirtyKV = dirtyKVCopy
		t.toRemove = toRemoveCopy

		return nil
	})
}

// View creates a view transaction for the cache.
func (t *TreeMemoryCache) View(cb func(TreeDatabaseTransaction) error) error {
	return t.underlyingStore.View(func(underlyingTx TreeDatabaseTransaction) error {
		t.lock.RLock()

		dirtyCopy := make(map[chainhash.Hash]Node)
		dirtyKVCopy := make(map[chainhash.Hash]chainhash.Hash)
		toRemoveCopy := make(map[chainhash.Hash]struct{})

		for k, v := range t.dirty {
			dirtyCopy[k] = v
		}

		for k, v := range t.dirtyKV {
			dirtyKVCopy[k] = v
		}

		for k := range t.toRemove {
			toRemoveCopy[k] = struct{}{}
		}

		t.lock.RUnlock()

		tx := &TreeMemoryCacheTransaction{
			underlyingTransaction: underlyingTx,
			root:                  t.root,
			dirty:                 dirtyCopy,
			dirtyKV:               dirtyKVCopy,
			toRemove:              toRemoveCopy,
			update:                false,
		}

		return cb(tx)
	})
}

// Flush flushes the transaction to the underlying tree.
func (t *TreeMemoryCache) Flush() error {
	return t.underlyingStore.Update(func(tx TreeDatabaseTransaction) error {
		for _, dirtyNode := range t.dirty {
			err := tx.SetNode(&dirtyNode)
			if err != nil {
				return err
			}
		}

		for c := range t.toRemove {
			err := tx.DeleteNode(c)
			if err != nil {
				return err
			}
		}

		root := t.root

		if !root.IsEqual(&primitives.EmptyTree) {
			rootNode, err := tx.GetNode(root)
			if err != nil {
				return err
			}

			err = tx.SetRoot(rootNode)
			if err != nil {
				return err
			}
		}

		for k, v := range t.dirtyKV {
			err := tx.Set(k, v)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// TreeMemoryCacheTransaction is a transaction wrapper on a CSMT that allows reading and writing from an underlying data store.
type TreeMemoryCacheTransaction struct {
	underlyingTransaction TreeDatabaseTransaction

	root chainhash.Hash

	toRemove map[chainhash.Hash]struct{}
	dirty map[chainhash.Hash]Node
	dirtyKV map[chainhash.Hash]chainhash.Hash
	update   bool
}

// Hash gets the root hash of the tree transaction cache.
func (t *TreeMemoryCacheTransaction) Hash() (*chainhash.Hash, error) {
	return &t.root, nil
}

// Root gets the current root of the transaction.
func (t *TreeMemoryCacheTransaction) Root() (*Node, error) {
	if t.root.IsEqual(&primitives.EmptyTree) {
		return nil, nil
	}

	if n, found := t.dirty[t.root]; found {
		return &n, nil
	}
	return t.underlyingTransaction.GetNode(t.root)
}

// SetRoot sets the root for the current transaction. If it does not exist in the cache or the underlying data store,
// add it to the cache.
func (t *TreeMemoryCacheTransaction) SetRoot(n *Node) error {
	if !t.update {
		return errors.New("set root called on View transaction")
	}

	nodeHash := n.GetHash()
	if _, found := t.dirty[nodeHash]; !found {
		if _, err := t.underlyingTransaction.GetNode(nodeHash); err != nil {
			err := t.SetNode(n)
			if err != nil {
				return err
			}
		}
	}
	t.root = nodeHash

	return nil
}

// NewNode creates a new node and adds it to the cache.
func (t *TreeMemoryCacheTransaction) NewNode(left *Node, right *Node, subtreeHash chainhash.Hash) (*Node, error) {
	if !t.update {
		return nil, errors.New("new node called on View transaction")
	}

	var leftHash *chainhash.Hash
	var rightHash *chainhash.Hash

	if left != nil {
		lh := left.GetHash()
		leftHash = &lh
	}

	if right != nil {
		rh := right.GetHash()
		rightHash = &rh
	}

	newNode := &Node{
		value: subtreeHash,
		left:  leftHash,
		right: rightHash,
	}
	t.dirty[subtreeHash] = *newNode
	return newNode, nil
}

// NewSingleNode creates a new node with only a single KV-pair in the subtree.
func (t *TreeMemoryCacheTransaction) NewSingleNode(key chainhash.Hash, value chainhash.Hash, subtreeHash chainhash.Hash) (*Node, error) {
	if !t.update {
		return nil, errors.New("new single node called on View transaction")
	}

	newNode := &Node{
		one:      true,
		oneKey:   &key,
		oneValue: &value,
		value:    subtreeHash,
	}
	t.dirty[subtreeHash] = *newNode
	return newNode, nil
}

// GetNode gets a node based on the subtree hash.
func (t *TreeMemoryCacheTransaction) GetNode(c chainhash.Hash) (*Node, error) {
	if n, found := t.dirty[c]; found {
		return &n, nil
	}
	return t.underlyingTransaction.GetNode(c)
}

// SetNode sets a node in the transaction.
func (t *TreeMemoryCacheTransaction) SetNode(n *Node) error {
	if !t.update {
		return errors.New("set node called on View transaction")
	}

	nodeHash := n.GetHash()
	t.dirty[nodeHash] = *n

	return nil
}

// DeleteNode deletes a node from the transaction.
func (t *TreeMemoryCacheTransaction) DeleteNode(c chainhash.Hash) error {
	if !t.update {
		return errors.New("delete node called on View transaction")
	}

	delete(t.dirty, c)

	t.toRemove[c] = struct{}{}

	return nil
}

// Get gets a value from the transaction.
func (t *TreeMemoryCacheTransaction) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	if val, found := t.dirtyKV[key]; found {
		return &val, nil
	} else {
		return t.underlyingTransaction.Get(key)
	}
}

// Set sets a value in the transaction.
func (t *TreeMemoryCacheTransaction) Set(key chainhash.Hash, val chainhash.Hash) error {
	if !t.update {
		return errors.New("set called on View transaction")
	}

	t.dirtyKV[key] = val

	return nil
}

var _ TreeDatabase = &TreeMemoryCache{}
