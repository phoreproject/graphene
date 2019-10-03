package csmt

import (
	"errors"
	"github.com/phoreproject/synapse/chainhash"
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
func NewTreeTransaction(underlyingStore TreeDatabase) (*TreeTransaction, error) {
	rootHash := EmptyTree
	root, err := underlyingStore.Root()
	if err != nil {
		return nil, err
	}
	if root != nil {
		rootHash = root.GetHash()
	}
	return &TreeTransaction{
		underlyingStore: underlyingStore,
		root: rootHash,
		dirty: make(map[chainhash.Hash]Node),
		dirtyKV: make(map[chainhash.Hash]chainhash.Hash),
		toRemove: make(map[chainhash.Hash]struct{}),
		valid: true,
	}, nil
}

// Root gets the current root of the transaction.
func (t *TreeTransaction) Root() (*Node, error) {
	if !t.valid {
		return nil, errors.New("root called on transaction already committed")
	}

	if t.root.IsEqual(&EmptyTree) {
		return nil, nil
	}

	if n, found := t.dirty[t.root]; found {
		return &n, nil
	}
	return t.underlyingStore.GetNode(t.root)
}

// SetRoot sets the root for the current transaction. If it does not exist in the cache or the underlying data store,
// add it to the cache.
func (t *TreeTransaction) SetRoot(n *Node) error {
	if !t.valid {
		return errors.New("SetRoot called on transaction already committed")
	}

	nodeHash := n.GetHash()
	if _, found := t.dirty[nodeHash]; !found {
		if _, err := t.underlyingStore.GetNode(nodeHash); err != nil {
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
func (t *TreeTransaction) NewNode(left *Node, right *Node, subtreeHash chainhash.Hash) (*Node, error) {
	if !t.valid {
		return nil, errors.New("NewNode called on transaction already committed")
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
		left: leftHash,
		right: rightHash,
	}
	t.dirty[subtreeHash] = *newNode
	return newNode, nil
}

// NewSingleNode creates a new node with only a single KV-pair in the subtree.
func (t *TreeTransaction) NewSingleNode(key chainhash.Hash, value chainhash.Hash, subtreeHash chainhash.Hash) (*Node, error) {
	if !t.valid {
		return nil, errors.New("NewSingleNode called on transaction already committed")
	}

	newNode := &Node{
		one: true,
		oneKey: &key,
		oneValue: &value,
		value: subtreeHash,
	}
	t.dirty[subtreeHash] = *newNode
	return newNode, nil
}

// GetNode gets a node based on the subtree hash.
func (t *TreeTransaction) GetNode(c chainhash.Hash) (*Node, error) {
	if !t.valid {
		return nil, errors.New("GetNode called on transaction already committed")
	}

	if n, found := t.dirty[c]; found {
		return &n, nil
	}
	return t.underlyingStore.GetNode(c)
}

// SetNode sets a node in the transaction.
func (t *TreeTransaction) SetNode(n *Node) error {
	if !t.valid {
		return errors.New("SetNode called on transaction already committed")
	}

	nodeHash := n.GetHash()
	t.dirty[nodeHash] = *n

	return nil
}

// DeleteNode deletes a node from the transaction.
func (t *TreeTransaction) DeleteNode(c chainhash.Hash) error {
	if !t.valid {
		return errors.New("DeleteNode called on transaction already committed")
	}

	delete(t.dirty, c)

	t.toRemove[c] = struct{}{}

	return nil
}

// Flush flushes the transaction to the underlying tree.
func (t *TreeTransaction) Flush() error {
	if !t.valid {
		return errors.New("flush called on transaction already committed")
	}

	for _, dirtyNode := range t.dirty {
		err := t.underlyingStore.SetNode(&dirtyNode)
		if err != nil {
			return err
		}
	}

	for c := range t.toRemove {
		err := t.underlyingStore.DeleteNode(c)
		if err != nil {
			return err
		}
	}

	root, err := t.Root()
	if err != nil {
		return err
	}

	err = t.underlyingStore.SetRoot(root)
	if err != nil {
		return err
	}

	for k, v := range t.dirtyKV {
		err := t.underlyingStore.Set(k, v)
		if err != nil {
			return err
		}
	}

	t.valid = false

	return nil
}

// Get gets a value from the transaction.
func (t *TreeTransaction) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	if !t.valid {
		return nil, errors.New("get called on transaction already committed")
	}

	if val, found := t.dirtyKV[key]; found {
		return &val, nil
	} else {
		return t.underlyingStore.Get(key)
	}
}

// Set sets a value in the transaction.
func (t *TreeTransaction) Set(key chainhash.Hash, val chainhash.Hash) error {
	if !t.valid {
		return errors.New("set called on transaction already committed")
	}

	t.dirtyKV[key] = val

	return nil
}

var _ TreeDatabase = &TreeTransaction{}