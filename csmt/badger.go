package csmt

import (
	"bytes"
	"github.com/dgraph-io/badger"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/pkg/errors"
)

// BadgerTreeDB is a tree database implemented on top of badger.
type BadgerTreeDB struct {
	db *badger.DB
}

var treeDBPrefix = []byte("tree-")
var treeKVPrefix = []byte("kv-")

func getTreeKey(key []byte) []byte {
	return append(treeDBPrefix, key...)
}

func getKVKey(key []byte) []byte {
	return append(treeKVPrefix, key...)
}

// SetRoot sets the root of the database.
func (b *BadgerTreeDB) SetRoot(n *Node) error {
	nodeHash := n.GetHash()

	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("root"), nodeHash[:])
	})
}

// NewNode creates a new node with the given left and right children and adds it to the database.
func (b *BadgerTreeDB) NewNode(left *Node, right *Node, subtreeHash chainhash.Hash) (*Node, error) {
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

	return newNode, b.SetNode(newNode)
}

// NewSingleNode creates a new single node and adds it to the database.
func (b *BadgerTreeDB) NewSingleNode(key chainhash.Hash, value chainhash.Hash, subtreeHash chainhash.Hash) (*Node, error) {
	n := &Node{
		one: true,
		oneKey: &key,
		oneValue: &value,
		value: subtreeHash,
	}

	return n, b.SetNode(n)
}

// GetNode gets a node from the database.
func (b *BadgerTreeDB) GetNode(nodeHash chainhash.Hash) (*Node, error) {
	nodeKey := getTreeKey(nodeHash[:])

	var n *Node

	err := b.db.View(func(txn *badger.Txn) error {
		nodeItem, err := txn.Get(nodeKey)
		if err != nil {
			return errors.Wrapf(err, "error getting node %s", nodeHash)
		}

		nodeSer, err := nodeItem.ValueCopy(nil)
		if err != nil {
			return err
		}

		n, err = DeserializeNode(nodeSer)
		if err != nil {
			return err
		}

		return nil
	})

	return n, err
}

// SetNode sets a node in the database.
func (b *BadgerTreeDB) SetNode(n *Node) error {
	nodeSer := n.Serialize()
	nodeHash := n.GetHash()
	nodeKey := getTreeKey(nodeHash[:])

	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(nodeKey, nodeSer)
	})
}

// DeleteNode deletes a node from the database.
func (b *BadgerTreeDB) DeleteNode(key chainhash.Hash) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(getTreeKey(key[:]))
	})
}

// Get gets a value from the key-value store.
func (b *BadgerTreeDB) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	var val chainhash.Hash

	err := b.db.View(func(txn *badger.Txn) error {
		valItem, err := txn.Get(getKVKey(key[:]))
		if err != nil {
			return err
		}

		_, err = valItem.ValueCopy(val[:])
		if err != nil {
			return err
		}

		return nil
	})
	return &val, err
}

// Set sets a value in the key-value store.
func (b *BadgerTreeDB) Set(key chainhash.Hash, value chainhash.Hash) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(getKVKey(key[:]), value[:])
	})
}

// Root gets the root node.
func (b *BadgerTreeDB) Root() (*Node, error) {
	var node *Node
	err := b.db.View(func(txn *badger.Txn) error {
		i, err := txn.Get([]byte("root"))
		if err != nil {
			return nil
		}

		rootHash, err := i.ValueCopy(nil)
		if err != nil {
			return err
		}

		if bytes.Equal(rootHash, EmptyTree[:]) {
			return nil
		}

		nodeKey, err := txn.Get(getTreeKey(rootHash))
		if err != nil {
			return err
		}

		nodeSer, err := nodeKey.ValueCopy(nil)
		if err != nil {
			return err
		}

		node, err = DeserializeNode(nodeSer)
		if err != nil {
			return err
		}

		return nil
	})
	return node, err
}

// NewBadgerTreeDB creates a new badger tree database from a badger database.
func NewBadgerTreeDB(db *badger.DB) *BadgerTreeDB {
	return &BadgerTreeDB{
		db: db,
	}
}

var _ TreeDatabase = &BadgerTreeDB{}