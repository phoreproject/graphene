package csmt

import (
	"github.com/phoreproject/synapse/chainhash"

	logger "github.com/sirupsen/logrus"
)

// TreeTransaction is a transaction wrapper on a CSMT that allows reading and writing from an underlying data store.
type TreeTransaction struct {
	underlyingStore TreeDatabase

	root chainhash.Hash

	// dirty are nodes that are marked as dirty
	dirty map[chainhash.Hash]Node
	dirtyKV map[chainhash.Hash]chainhash.Hash

	toRemove map[chainhash.Hash]struct{}

	valid bool
}

// NewTreeTransaction constructs a tree transaction that can be committed based on an underlying tree (probably on disk).
func NewTreeTransaction(underlyingStore TreeDatabase) TreeTransaction {
	rootHash := EmptyTree
	root := underlyingStore.Root()
	if root != nil {
		rootHash = root.GetHash()
	}
	return TreeTransaction{
		underlyingStore: underlyingStore,
		root: rootHash,
		dirty: make(map[chainhash.Hash]Node),
		dirtyKV: make(map[chainhash.Hash]chainhash.Hash),
		toRemove: make(map[chainhash.Hash]struct{}),
		valid: true,
	}
}

// Root gets the current root of the transaction.
func (t *TreeTransaction) Root() Node {
	if n, found := t.dirty[t.root]; found {
		return n
	} else {
		if n, found := t.underlyingStore.GetNode(t.root); found {
			return n
		} else {
			return nil
		}
	}
}

// SetRoot sets the root for the current transaction. If it does not exist in the cache or the underlying data store,
// add it to the cache.
func (t *TreeTransaction) SetRoot(n Node) {
	if !t.valid {
		logger.Warn("SetRoot called on transaction already committed")
		return
	}

	nodeHash := n.GetHash()
	if _, found := t.dirty[nodeHash]; !found {
		if _, found := t.underlyingStore.GetNode(nodeHash); !found {
			t.SetNode(n)
		}
	}
	t.root = nodeHash
}

// NewNode creates a new node and adds it to the cache.
func (t *TreeTransaction) NewNode(left Node, right Node, subtreeHash chainhash.Hash) Node {
	if !t.valid {
		logger.Warn("NewNode called on transaction already committed")
		return nil
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

	newNode := &InMemoryNode{
		value: subtreeHash,
		left: leftHash,
		right: rightHash,
	}
	t.dirty[subtreeHash] = newNode
	return newNode
}

// NewSingleNode creates a new node with only a single KV-pair in the subtree.
func (t *TreeTransaction) NewSingleNode(key chainhash.Hash, value chainhash.Hash, subtreeHash chainhash.Hash) Node {
	if !t.valid {
		logger.Warn("NewSingleNode called on transaction already committed")
		return nil
	}

	newNode := &InMemoryNode{
		one: true,
		oneKey: &key,
		oneValue: &value,
		value: subtreeHash,
	}
	t.dirty[subtreeHash] = newNode
	return newNode
}

// GetNode gets a node based on the subtree hash.
func (t *TreeTransaction) GetNode(c chainhash.Hash) (Node, bool) {
	if n, found := t.dirty[c]; found {
		return n, true
	} else {
		if n, found := t.underlyingStore.GetNode(c); found {
			return n, true
		} else {
			return nil, false
		}
	}
}

// SetNode sets a node in the transaction.
func (t *TreeTransaction) SetNode(n Node) {
	if !t.valid {
		logger.Warn("SetNode called on transaction already committed")
		return
	}

	t.dirty[n.GetHash()] = n
}

// DeleteNode deletes a node from the transaction.
func (t *TreeTransaction) DeleteNode(c chainhash.Hash) {
	if !t.valid {
		logger.Warn("DeleteNode called on transaction already committed")
		return
	}

	delete(t.dirty, c)

	t.toRemove[c] = struct{}{}
}

// Flush flushes the transaction to the underlying tree.
func (t *TreeTransaction) Flush() {
	t.valid = false

	for _, dirtyNode := range t.dirty {
		t.underlyingStore.SetNode(dirtyNode)
	}

	for c := range t.toRemove {
		t.underlyingStore.DeleteNode(c)
	}

	t.underlyingStore.SetRoot(t.Root())

	for k, v := range t.dirtyKV {
		t.underlyingStore.Set(k, v)
	}
}

// Get gets a value from the transaction.
func (t *TreeTransaction) Get(key chainhash.Hash) (*chainhash.Hash, bool) {
	if val, found := t.dirtyKV[key]; found {
		return &val, true
	} else {
		return t.underlyingStore.Get(key)
	}
}

// Set sets a value in the transaction.
func (t *TreeTransaction) Set(key chainhash.Hash, val chainhash.Hash) {
	if !t.valid {
		logger.Warn("SetNode called on transaction already committed")
		return
	}

	t.dirtyKV[key] = val
}

var _ TreeDatabase = &TreeTransaction{}