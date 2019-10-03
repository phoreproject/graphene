package csmt

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/phoreproject/synapse/chainhash"
	"io"
)

// InMemoryTreeDB is a tree stored in memory.
type InMemoryTreeDB struct {
	root chainhash.Hash
	nodes map[chainhash.Hash]Node
	store map[chainhash.Hash]chainhash.Hash
}

// NewInMemoryTreeDB creates a new in-memory tree database.
func NewInMemoryTreeDB() *InMemoryTreeDB {
	return &InMemoryTreeDB{
		root: EmptyTree,
		nodes: make(map[chainhash.Hash]Node),
		store: make(map[chainhash.Hash]chainhash.Hash),
	}
}

// GetNode gets a node from the tree database.
func (i *InMemoryTreeDB) GetNode(nodeHash chainhash.Hash) (*Node, error) {
	if n, found := i.nodes[nodeHash]; found {
		return &n, nil
	} else {
		return nil, fmt.Errorf("could not find node with hash %s", nodeHash)
	}
}

// SetNode sets a node in the database. The node passed MUST be an Node
func (i *InMemoryTreeDB) SetNode(n *Node) error {
	nodeHash := n.GetHash()
	i.nodes[nodeHash] = *n

	return nil
}

// DeleteNode deletes a node if it exists.
func (i *InMemoryTreeDB) DeleteNode(h chainhash.Hash) error {
	delete(i.nodes, h)

	return nil
}

// Root gets the root of the tree.
func (i *InMemoryTreeDB) Root() (*Node, error) {
	if n, found := i.nodes[i.root]; found {
		return &n, nil
	} else {
		return nil, nil
	}
}

// SetRoot sets the root of the tree.
func (i *InMemoryTreeDB) SetRoot(n *Node) error {
	nodeHash := n.GetHash()
	if _, found := i.nodes[nodeHash]; !found {
		err := i.SetNode(n)
		if err != nil {
			return err
		}
	}
	i.root = nodeHash

	return nil
}

// NewNode creates a new empty node.
func (i *InMemoryTreeDB) NewNode(left *Node, right *Node, subtreeHash chainhash.Hash) (*Node, error) {
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
	i.nodes[subtreeHash] = *newNode
	return newNode, nil
}

// NewSingleNode creates a new node with only one key-value pair.
func (i *InMemoryTreeDB) NewSingleNode(key chainhash.Hash, value chainhash.Hash, subtreeHash chainhash.Hash) (*Node, error) {
	newNode := &Node{
		one: true,
		oneKey: &key,
		oneValue: &value,
		value: subtreeHash,
	}
	i.nodes[subtreeHash] = *newNode
	return newNode, nil
}

// Empty checks if the node is empty.
func (i *Node) Empty() bool {
	return i == nil
}

// Get gets a value from the key-value store.
func (i *InMemoryTreeDB) Get(k chainhash.Hash) (*chainhash.Hash, error) {
	if v, found := i.store[k]; found {
		return &v, nil
	}
	return nil, nil
}

// Set sets a value in the key-value store.
func (i *InMemoryTreeDB) Set(k chainhash.Hash, v chainhash.Hash) error {
	i.store[k] = v

	return nil
}

// Node is a node of the in-memory tree database.
type Node struct {
	value    chainhash.Hash
	one      bool
	oneKey   *chainhash.Hash
	oneValue *chainhash.Hash
	left     *chainhash.Hash
	right    *chainhash.Hash
}

const (
	// FlagSingle designates that this node is a single node.
	FlagSingle = iota

	// FlagLeft designates that this node has a left branch.
	FlagLeft

	// FlagRight designates that this node has a right branch.
	FlagRight

	// FlagBoth designates that this node has both a left and right branch.
	FlagBoth
)

func readHash(r io.Reader) (*chainhash.Hash, error) {
	var value chainhash.Hash
	n, err := r.Read(value[:])
	if err != nil {
		return nil, err
	}
	if n != 32 {
		return nil, errors.New("expected to read 32 bytes from node")
	}
	return &value, nil
}

// DeserializeNode deserializes a node from disk.
func DeserializeNode(b []byte) (*Node, error) {
	r := bytes.NewBuffer(b)

	flag, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	hash, err := readHash(r)
	if err != nil {
		return nil, err
	}

	switch flag {
	case FlagSingle:
		key, err := readHash(r)
		if err != nil {
			return nil, err
		}
		val, err := readHash(r)
		if err != nil {
			return nil, err
		}
		return &Node{
			value:    *hash,
			one:      true,
			oneKey:   key,
			oneValue: val,
		}, nil
	case FlagLeft:
		left, err := readHash(r)
		if err != nil {
			return nil, err
		}
		return &Node{
			value:    *hash,
			left: left,
		}, nil
	case FlagRight:
		right, err := readHash(r)
		if err != nil {
			return nil, err
		}
		return &Node{
			value:    *hash,
			right: right,
		}, nil
	case FlagBoth:
		left, err := readHash(r)
		if err != nil {
			return nil, err
		}
		right, err := readHash(r)
		if err != nil {
			return nil, err
		}
		return &Node{
			value:    *hash,
			left: left,
			right: right,
		}, nil
	default:
		return nil, errors.New("unexpected flag")
	}
}

// Serialize gets the node as a byte representation.
func (i *Node) Serialize() []byte {
	buf := bytes.NewBuffer(nil)
	var flag byte
	if i.one {
		flag = FlagSingle
	} else if i.left != nil && i.right != nil {
		flag = FlagBoth
	} else if i.left != nil {
		flag = FlagLeft
	} else if i.right != nil {
		flag = FlagRight
	} else {
		panic("improper node (not single and no left/right)")
	}
	buf.WriteByte(flag)
	buf.Write(i.value[:])
	if i.one {
		buf.Write(i.oneKey[:])
		buf.Write(i.oneValue[:])
	} else {
		if i.left != nil {
			buf.Write(i.left[:])
		}
		if i.right != nil {
			buf.Write(i.right[:])
		}
	}
	return buf.Bytes()
}

// GetHash gets the current hash from memory.
func (i *Node) GetHash() chainhash.Hash {
	return i.value
}

// Left gets the left node in memory.
func (i *Node) Left() *chainhash.Hash {
	return i.left
}

// Right gets the right node in memory.
func (i *Node) Right() *chainhash.Hash {
	return i.right
}

// IsSingle checks if there is only a single key in the subtree.
func (i *Node) IsSingle() bool {
	return i.one
}

// GetSingleKey gets the only key in the subtree.
func (i *Node) GetSingleKey() chainhash.Hash {
	if i.oneKey != nil {
		return *i.oneKey
	} else {
		return chainhash.Hash{}
	}
}

// GetSingleValue gets the only value in the subtree.
func (i *Node) GetSingleValue() chainhash.Hash {
	if i.oneValue != nil {
		return *i.oneValue
	} else {
		return chainhash.Hash{}
	}
}

var _ TreeDatabase = &InMemoryTreeDB{}


