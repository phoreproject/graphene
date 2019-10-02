package csmt

import "github.com/phoreproject/synapse/chainhash"

// InMemoryTreeDB is a tree stored in memory.
type InMemoryTreeDB struct {
	root chainhash.Hash
	nodes map[chainhash.Hash]InMemoryNode
}

// NewInMemoryTreeDB creates a new in-memory tree database.
func NewInMemoryTreeDB() *InMemoryTreeDB {
	return &InMemoryTreeDB{
		root: EmptyTree,
		nodes: make(map[chainhash.Hash]InMemoryNode),
	}
}

// GetNode gets a node from the tree database.
func (i *InMemoryTreeDB) GetNode(nodeHash chainhash.Hash) (Node, bool) {
	if n, found := i.nodes[nodeHash]; found {
		return &n, false
	} else {
		return nil, true
	}
}

// SetNode sets a node in the database. The node passed MUST be an InMemoryNode
func (i *InMemoryTreeDB) SetNode(n Node) {
	if imn, ok := n.(*InMemoryNode); ok {
		i.nodes[n.GetHash()] = *imn
	} else {
		panic("InMemoryTreeDB.SetNode(Node) must be called with an in-memory node.")
	}
}

// DeleteNode deletes a node if it exists.
func (i *InMemoryTreeDB) DeleteNode(h chainhash.Hash) {
	delete(i.nodes, h)
}

// Root gets the root of the tree.
func (i *InMemoryTreeDB) Root() Node {
	if n, found := i.nodes[i.root]; found {
		return &n
	} else {
		return nil
	}
}

// SetRoot sets the root of the tree.
func (i *InMemoryTreeDB) SetRoot(n Node) {
	nodeHash := n.GetHash()
	if _, found := i.nodes[nodeHash]; !found {
		i.SetNode(n)
	}
	i.root = nodeHash
}

// NewNode creates a new empty node.
func (i *InMemoryTreeDB) NewNode(left Node, right Node, subtreeHash chainhash.Hash) Node {
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
	i.nodes[subtreeHash] = *newNode
	return newNode
}

// NewSingleNode creates a new node with only one key-value pair.
func (i *InMemoryTreeDB) NewSingleNode(key chainhash.Hash, value chainhash.Hash, subtreeHash chainhash.Hash) Node {
	newNode := &InMemoryNode{
		one: true,
		oneKey: &key,
		oneValue: &value,
		value: subtreeHash,
	}
	i.nodes[subtreeHash] = *newNode
	return newNode
}

// InMemoryNode is a node of the in-memory tree database.
type InMemoryNode struct {
	value    chainhash.Hash
	one      bool
	oneKey   *chainhash.Hash
	oneValue *chainhash.Hash
	left     *chainhash.Hash
	right    *chainhash.Hash
}

// GetHash gets the current hash from memory.
func (i *InMemoryNode) GetHash() chainhash.Hash {
	return i.value
}

// Left gets the left node in memory.
func (i *InMemoryNode) Left() *chainhash.Hash {
	return i.left
}

// Right gets the right node in memory.
func (i *InMemoryNode) Right() *chainhash.Hash {
	return i.right
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

// Empty checks if the node is empty.
func (i *InMemoryNode) Empty() bool {
	return i == nil
}

// InMemoryKVStore is a key-value store in memory.
type InMemoryKVStore struct {
	store map[chainhash.Hash]chainhash.Hash
}

// NewInMemoryKVStore constructs a new key-value store in memory.
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


