package csmt

import (
	"bytes"

	"github.com/dgraph-io/badger"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/pkg/errors"
)

// BadgerTreeDB is a tree database implemented on top of badger.
type BadgerTreeDB struct {
	db *badger.DB
}

// Hash gets the hash of the root.
func (b *BadgerTreeDB) Hash() (*chainhash.Hash, error) {
	out := primitives.EmptyTree
	err := b.View(func(transaction TreeDatabaseTransaction) error {
		h, err := transaction.Hash()
		if err != nil {
			return err
		}
		out = *h
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &out, nil
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
func (b *BadgerTreeTransaction) SetRoot(n *Node) error {
	nodeHash := n.GetHash()

	return b.tx.Set([]byte("root"), nodeHash[:])
}

// NewNode creates a new node with the given left and right children and adds it to the database.
func (b *BadgerTreeTransaction) NewNode(left *Node, right *Node, subtreeHash chainhash.Hash) (*Node, error) {
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

	return newNode, b.SetNode(newNode)
}

// NewSingleNode creates a new single node and adds it to the database.
func (b *BadgerTreeTransaction) NewSingleNode(key chainhash.Hash, value chainhash.Hash, subtreeHash chainhash.Hash) (*Node, error) {
	n := &Node{
		one:      true,
		oneKey:   &key,
		oneValue: &value,
		value:    subtreeHash,
	}

	return n, b.SetNode(n)
}

// GetNode gets a node from the database.
func (b *BadgerTreeTransaction) GetNode(nodeHash chainhash.Hash) (*Node, error) {
	nodeKey := getTreeKey(nodeHash[:])

	nodeItem, err := b.tx.Get(nodeKey)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting node %s", nodeHash)
	}

	nodeSer, err := nodeItem.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return DeserializeNode(nodeSer)
}

// SetNode sets a node in the database.
func (b *BadgerTreeTransaction) SetNode(n *Node) error {
	nodeSer := n.Serialize()
	nodeHash := n.GetHash()
	nodeKey := getTreeKey(nodeHash[:])

	return b.tx.Set(nodeKey, nodeSer)
}

// DeleteNode deletes a node from the database.
func (b *BadgerTreeTransaction) DeleteNode(key chainhash.Hash) error {
	return b.tx.Delete(getTreeKey(key[:]))
}

// Get gets a value from the key-value store.
func (b *BadgerTreeTransaction) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	var val chainhash.Hash

	valItem, err := b.tx.Get(getKVKey(key[:]))
	if err != nil {
		return nil, err
	}

	_, err = valItem.ValueCopy(val[:])
	if err != nil {
		return nil, err
	}

	return &val, nil
}

// Set sets a value in the key-value store.
func (b *BadgerTreeTransaction) Set(key chainhash.Hash, value chainhash.Hash) error {
	return b.tx.Set(getKVKey(key[:]), value[:])
}

// Root gets the root node.
func (b *BadgerTreeTransaction) Root() (*Node, error) {
	i, err := b.tx.Get([]byte("root"))
	if err != nil {
		return nil, nil
	}

	rootHash, err := i.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	if bytes.Equal(rootHash, primitives.EmptyTree[:]) {
		return nil, nil
	}

	nodeKey, err := b.tx.Get(getTreeKey(rootHash))
	if err != nil {
		return nil, err
	}

	nodeSer, err := nodeKey.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return DeserializeNode(nodeSer)
}

// BadgerTreeTransaction represents a badger transaction.
type BadgerTreeTransaction struct {
	tx *badger.Txn
}

// Hash gets the hash of the root.
func (b *BadgerTreeTransaction) Hash() (*chainhash.Hash, error) {
	i, err := b.tx.Get([]byte("root"))
	if err != nil {
		return &primitives.EmptyTree, nil
	}

	rootHash, err := i.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return chainhash.NewHash(rootHash)
}

// Update updates the database.
func (b *BadgerTreeDB) Update(callback func(TreeDatabaseTransaction) error) error {
	badgerTx := b.db.NewTransaction(true)

	err := callback(&BadgerTreeTransaction{badgerTx})
	if err != nil {
		badgerTx.Discard()
		return err
	}
	return badgerTx.Commit()
}

// View creates a read-only transaction for the database.
func (b *BadgerTreeDB) View(callback func(TreeDatabaseTransaction) error) error {
	badgerTx := b.db.NewTransaction(false)

	err := callback(&BadgerTreeTransaction{badgerTx})
	if err != nil {
		badgerTx.Discard()
		return err
	}
	return badgerTx.Commit()
}

// NewBadgerTreeDB creates a new badger tree database from a badger database.
func NewBadgerTreeDB(db *badger.DB) *BadgerTreeDB {
	return &BadgerTreeDB{
		db: db,
	}
}

var _ TreeDatabase = &BadgerTreeDB{}
