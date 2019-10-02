package csmt

import (
	"github.com/phoreproject/synapse/chainhash"
)

// TreeDatabase is a database that keeps track of a tree.
type TreeDatabase interface {
	// Root gets the root node.
	Root() Node

	// SetRoot sets the root node.
	SetRoot(Node)

	// NewNode creates a new node.
	NewNode() Node

	// NewNodeWithHash creates a new node with the given initial subtree hash.
	NewNodeWithHash(subtreeHash chainhash.Hash) Node

	// NewSingleNode creates a new node that represents a subtree with only a single key.
	NewSingleNode(key chainhash.Hash, value chainhash.Hash, subtreeHash chainhash.Hash) Node
}

// KVStore is a database that associates keys with values.
type KVStore interface {
	// Get a value from the kv store.
	Get(chainhash.Hash) (*chainhash.Hash, bool)

	// Set a value in the kv store.
	Set(chainhash.Hash, chainhash.Hash)
}

// Node is a node in the tree database.
type Node interface {
	// GetHash gets the current hash of the subtree
	GetHash() chainhash.Hash

	// SetHash sets the current hash of the subtree.
	SetHash(chainhash.Hash)

	// Left gets the node on the left side or returns nil if there is no node on the left.
	Left() Node

	// SetLeft sets the node on the left side.
	SetLeft(Node)

	// Right gets the node on the right side or returns nil if there is no node on the right.
	Right() Node

	// SetRight sets the node on the right side.
	SetRight(Node)

	// IsSingle returns true if there is only one key in this subtree.
	IsSingle() bool

	// GetSingleKey gets the key of the only key in this subtree. Undefined if not a single node.
	GetSingleKey() chainhash.Hash

	// GetSingleValue gets the value of the only key in this subtree. Undefined if not a single node.
	GetSingleValue() chainhash.Hash

	// SetSingleValue sets the value of the only key in this subtree.
	SetSingleValue(chainhash.Hash)

	// Empty checks if the node is empty.
	Empty() bool
}

// MemoryNode represents a node in the merkle tree.
type MemoryNode struct {
	Value    chainhash.Hash
	One      bool
	OneKey   *chainhash.Hash
	OneValue *chainhash.Hash
	Left     *MemoryNode
	Right    *MemoryNode
}

// Copy returns a deep copy of the tree.
func (n *MemoryNode) Copy() *MemoryNode {
	if n == nil {
		return nil
	}

	newNode := &MemoryNode{
		Value: n.Value,
		One:   n.One,
	}

	if n.OneKey != nil {
		newNode.OneKey = &chainhash.Hash{}
		copy(newNode.OneKey[:], n.OneKey[:])
	}

	if n.OneValue != nil {
		newNode.OneValue = &chainhash.Hash{}
		copy(newNode.OneValue[:], n.OneValue[:])
	}

	if n.Left != nil {
		newNode.Left = n.Left.Copy()
	}

	if n.Right != nil {
		newNode.Right = n.Right.Copy()
	}

	return newNode
}

// Tree is a wr
type Tree struct {
	tree TreeDatabase
	kv   KVStore
}

// NewTree creates a MemoryTree
func NewTree(d TreeDatabase, store KVStore) Tree {
	return Tree{
		tree: d,
		kv:   store,
	}
}


// Hash get the root hash
func (t *Tree) Hash() chainhash.Hash {
	r := t.tree.Root()
	if r.Empty() {
		return emptyTrees[255]
	}
	return r.GetHash()
}

// Set inserts/updates a value
func (t *Tree) Set(key chainhash.Hash, value chainhash.Hash) {
	// if the t is empty, insert at the root

	hk := chainhash.HashH(key[:])

	t.tree.SetRoot(insertIntoTree(t.tree, t.tree.Root(), hk, value, 255))

	t.kv.Set(key, value)
}

// SetWithWitness returns an update witness and sets the value in the tree.
func (t *Tree) SetWithWitness(key chainhash.Hash, value chainhash.Hash) *UpdateWitness {
	uw := GenerateUpdateWitness(t.tree, t.kv, key, value)
	t.Set(key, value)

	return &uw
}

// Prove proves a key in the tree.
func (t *Tree) Prove(key chainhash.Hash) *VerificationWitness {
	vw := GenerateVerificationWitness(t.tree, t.kv, key)
	return &vw
}


// Get gets a value from the tree.
func (t *Tree) Get(key chainhash.Hash) *chainhash.Hash {
	h, found := t.kv.Get(key)
	if !found {
		return &emptyHash
	}
	return h
}