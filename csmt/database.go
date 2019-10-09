package csmt

import (
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// TreeDatabase provides functions to update or view parts of the tree.
type TreeDatabase interface {
	Update(func (TreeDatabaseTransaction) error) error

	View(func (TreeDatabaseTransaction) error) error
}

// TreeDatabaseTransaction is a transaction interface to access or update the tree.
type TreeDatabaseTransaction interface {
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

// Tree represents a state tree.
type Tree struct {
	db TreeDatabase
}

// NewTree creates a MemoryTree
func NewTree(d TreeDatabase) Tree {
	return Tree{
		db: d,
	}
}

// Update creates a transaction for the tree.
func (t *Tree) Update(cb func (TreeTransaction) error) error {
	return t.db.Update(func(tx TreeDatabaseTransaction) error {
		return cb(TreeTransaction{tx, true})
	})
}

// View creates a view-only transaction for the tree.
func (t *Tree) View(cb func (TreeTransaction) error) error {
	return t.db.View(func(tx TreeDatabaseTransaction) error {
		return cb(TreeTransaction{tx, false})
	})
}

// Hash gets the hash of the tree.
func (t *Tree) Hash() (chainhash.Hash, error) {
	out := primitives.EmptyTree
	err := t.View(func(tx TreeTransaction) error {
		h, err := tx.Hash()
		if err != nil {
			return err
		}
		out = *h
		return nil
	})
	return out, err
}

// TreeTransaction is a wr
type TreeTransaction struct {
	tx TreeDatabaseTransaction
	update bool
}

// Hash get the root hash
func (t *TreeTransaction) Hash() (*chainhash.Hash, error) {
	treeHash := primitives.EmptyTree

	r, err := t.tx.Root()
	if err != nil {
		return nil, err
	}
	if r == nil || r.Empty() {
		return &treeHash, nil
	}
	treeHash = r.GetHash()

	return &treeHash, err
}

// Set inserts/updates a value
func (t *TreeTransaction) Set(key chainhash.Hash, value chainhash.Hash) error {
	// if the t is empty, insert at the root

	hk := chainhash.HashH(key[:])

	root, err := t.tx.Root()
	if err != nil {
		return err
	}

	n, err := insertIntoTree(t.tx, root, hk, value, 255)
	if err != nil {
		return err
	}

	err = t.tx.SetRoot(n)
	if err != nil {
		return err
	}

	return t.tx.Set(key, value)
}

// SetWithWitness returns an update witness and sets the value in the tree.
func (t *TreeTransaction) SetWithWitness(key chainhash.Hash, value chainhash.Hash) (*primitives.UpdateWitness, error) {
	uw, err := GenerateUpdateWitness(t.tx, key, value)
	if err != nil {
		return nil, err
	}

	err = t.Set(key, value)

	return uw, err
}

// Prove proves a key in the tree.
func (t *TreeTransaction) Prove(key chainhash.Hash) (*primitives.VerificationWitness, error) {
	vw, err := GenerateVerificationWitness(t.tx, key)
	if err != nil {
		return nil, err
	}

	return vw, err
}


// Get gets a value from the tree.
func (t *TreeTransaction) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	h, err := t.tx.Get(key)
	if err != nil {
		return nil, err
	}
	if h == nil {
		out := emptyHash
		h = &out
	}

	return h, err
}