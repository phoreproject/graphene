package csmt

import "github.com/phoreproject/synapse/chainhash"

// InMemoryTreeDB is a tree stored in memory.
type InMemoryTreeDB struct {
	root *InMemoryNode
}

// NewInMemoryTreeDB creates a new in-memory tree database.
func NewInMemoryTreeDB() *InMemoryTreeDB {
	return &InMemoryTreeDB{
		root: nil,
	}
}

// Root gets the root of the tree.
func (i *InMemoryTreeDB) Root() Node {
	return i.root
}

// SetRoot sets the root of the tree.
func (i *InMemoryTreeDB) SetRoot(n Node) {
	imn, _ := n.(*InMemoryNode)
	i.root = imn
}

// NewNode creates a new empty node.
func (InMemoryTreeDB) NewNode() Node {
	return &InMemoryNode{}
}

// NewNodeWithHash creates a new node with a specific hash.
func (i *InMemoryTreeDB) NewNodeWithHash(subtreeHash chainhash.Hash) Node {
	return &InMemoryNode{
		value: subtreeHash,
	}
}

// NewSingleNode creates a new node with only one key-value pair.
func (i *InMemoryTreeDB) NewSingleNode(key chainhash.Hash, value chainhash.Hash, subtreeHash chainhash.Hash) Node {
	return &InMemoryNode{
		one: true,
		oneKey: &key,
		oneValue: &value,
		value: subtreeHash,
	}
}

// InMemoryNode is a node of the in-memory tree database.
type InMemoryNode struct {
	value    chainhash.Hash
	one      bool
	oneKey   *chainhash.Hash
	oneValue *chainhash.Hash
	left     *InMemoryNode
	right    *InMemoryNode
}

// GetHash gets the current hash from memory.
func (i *InMemoryNode) GetHash() chainhash.Hash {
	return i.value
}

// SetHash sets the current hash in memory.
func (i *InMemoryNode) SetHash(v chainhash.Hash) {
	i.value = v
}

// Left gets the left node in memory.
func (i *InMemoryNode) Left() Node {
	return i.left
}

// SetLeft sets the left node in memory.
func (i *InMemoryNode) SetLeft(n Node) {
	imn, _ := n.(*InMemoryNode)
	i.left = imn
}

// Right gets the right node in memory.
func (i *InMemoryNode) Right() Node {
	return i.right
}

// SetRight sets the right node in memory.
func (i *InMemoryNode) SetRight(n Node) {
	imn, _ := n.(*InMemoryNode)
	i.right = imn
}

// IsSingle checks if there is only a single key in the subtree.
func (i *InMemoryNode) IsSingle() bool {
	return i.one
}

// GetSingleKey gets the only key in the subtree.
func (i *InMemoryNode) GetSingleKey() chainhash.Hash {
	if i.oneKey != nil {
		return *i.oneKey
	} else {
		return chainhash.Hash{}
	}
}

// GetSingleValue gets the only value in the subtree.
func (i *InMemoryNode) GetSingleValue() chainhash.Hash {
	if i.oneValue != nil {
		return *i.oneValue
	} else {
		return chainhash.Hash{}
	}
}

// SetSingleValue sets the only value in the subtree.
func (i *InMemoryNode) SetSingleValue(v chainhash.Hash) {
	*i.oneValue = v
}

// Empty checks if the node is empty.
func (i *InMemoryNode) Empty() bool {
	return i == nil
}

// InMemoryKVStore is a key-value store in memory.
type InMemoryKVStore struct {
	store map[chainhash.Hash]chainhash.Hash
}

func NewInMemoryKVStore() *InMemoryKVStore {
	return &InMemoryKVStore{
		store: make(map[chainhash.Hash]chainhash.Hash),
	}
}

// Get gets a value from the key-value store.
func (i *InMemoryKVStore) Get(k chainhash.Hash) (*chainhash.Hash, bool) {
	if v, found := i.store[k]; found {
		return &v, true
	}
	return nil, false
}

// Set sets a value in the key-value store.
func (i *InMemoryKVStore) Set(k chainhash.Hash, v chainhash.Hash) {
	i.store[k] = v
}

var _ KVStore = &InMemoryKVStore{}
var _ TreeDatabase = &InMemoryTreeDB{}
var _ Node = &InMemoryNode{}


