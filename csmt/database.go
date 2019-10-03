package csmt

import (
	"github.com/phoreproject/synapse/chainhash"
)

// TreeDatabase is a database that keeps track of a tree.
type TreeDatabase interface {
	// Root gets the root node.
	Root() (*Node, error)

	// SetRoot sets the root node.
	SetRoot(*Node) error

	// NewNode creates a new node, adds it to the tree database, and returns it.
	NewNode(left *Node, right *Node, subtreeHash chainhash.Hash) (*Node, error)

	// NewSingleNode creates a new node that represents a subtree with only a single key.
	NewSingleNode(key chainhash.Hash, value chainhash.Hash, subtreeHash chainhash.Hash) (*Node, error)

	// GetNode gets a node from the database.
	GetNode(chainhash.Hash) (*Node, error)

	// SetNode sets a node in the database.
	SetNode(*Node) error

	// DeleteNode deletes a node from the database.
	DeleteNode(chainhash.Hash) error

	// Get a value from the kv store.
	Get(chainhash.Hash) (*chainhash.Hash, error)

	// Set a value in the kv store.
	Set(chainhash.Hash, chainhash.Hash) error
}

// Tree is a wr
type Tree struct {
	tree TreeDatabase
}

// NewTree creates a MemoryTree
func NewTree(d TreeDatabase) Tree {
	return Tree{
		tree: d,
	}
}


// Hash get the root hash
func (t *Tree) Hash() (*chainhash.Hash, error) {
	r, err := t.tree.Root()
	if err != nil {
		return nil, err
	}
	if r == nil || r.Empty() {
		return &emptyTrees[255], nil
	}
	h := r.GetHash()
	return &h, nil
}

// Set inserts/updates a value
func (t *Tree) Set(key chainhash.Hash, value chainhash.Hash) error {
	// if the t is empty, insert at the root

	hk := chainhash.HashH(key[:])

	root, err := t.tree.Root()
	if err != nil {
		return err
	}

	n, err := insertIntoTree(t.tree, root, hk, value, 255)
	if err != nil {
		return err
	}

	err = t.tree.SetRoot(n)
	if err != nil {
		return err
	}

	return t.tree.Set(key, value)
}

// SetWithWitness returns an update witness and sets the value in the tree.
func (t *Tree) SetWithWitness(key chainhash.Hash, value chainhash.Hash) (*UpdateWitness, error) {
	uw, err := GenerateUpdateWitness(t.tree, key, value)
	if err != nil {
		return nil, err
	}

	err = t.Set(key, value)
	if err != nil {
		return nil, err
	}

	return uw, nil
}

// Prove proves a key in the tree.
func (t *Tree) Prove(key chainhash.Hash) (*VerificationWitness, error) {
	vw, err := GenerateVerificationWitness(t.tree, key)
	if err != nil {
		return nil, err
	}
	return vw, nil
}


// Get gets a value from the tree.
func (t *Tree) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	h, err := t.tree.Get(key)
	if err != nil {
		return nil, err
	}
	if h == nil {
		return &emptyHash, nil
	}
	return h, nil
}