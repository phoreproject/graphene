package csmt

import (
	"github.com/phoreproject/synapse/chainhash"
)

// Node represents a node in the merkle tree.
type Node struct {
	Value    chainhash.Hash
	One      bool
	OneKey   *chainhash.Hash
	OneValue *chainhash.Hash
	Left     *Node
	Right    *Node
}

// Copy returns a deep copy of the tree.
func (n *Node) Copy() *Node {
	if n == nil {
		return nil
	}

	newNode := &Node{
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

// Tree is Compact Sparse Merkle Tree
// It implements interface SMT
type Tree struct {
	root      *Node
	datastore map[chainhash.Hash]chainhash.Hash
}

func combineHashes(left *chainhash.Hash, right *chainhash.Hash) chainhash.Hash {
	return chainhash.HashH(append(left[:], right[:]...))
}

// NewTree creates a Tree
func NewTree() Tree {
	return Tree{
		root:      nil,
		datastore: map[chainhash.Hash]chainhash.Hash{},
	}
}

var emptyHash = chainhash.Hash{}
var emptyTrees [256]chainhash.Hash

// EmptyTree is the hash of an empty tree.
var EmptyTree = chainhash.Hash{}

func init() {
	emptyTrees[0] = emptyHash
	for i := range emptyTrees[1:] {
		emptyTrees[i+1] = combineHashes(&emptyTrees[i], &emptyTrees[i])
	}

	EmptyTree = emptyTrees[255]
}

// Hash get the root hash
func (t *Tree) Hash() chainhash.Hash {
	if t.root == nil {
		return emptyTrees[255]
	}
	return t.root.Value
}

// Set inserts/updates a value
func (t *Tree) Set(key chainhash.Hash, value chainhash.Hash) {
	// if the t is empty, insert at the root

	hk := chainhash.HashH(key[:])

	t.root = t.insert(t.root, &hk, &value, 255)

	t.datastore[key] = value
}

// SetWithWitness returns an update witness and sets the value in the tree.
func (t *Tree) SetWithWitness(key chainhash.Hash, value chainhash.Hash) *UpdateWitness {
	oldVal := t.Get(key)
	t.Set(key, *oldVal)
	uw := GenerateUpdateWitness(t, key, value)
	t.Set(key, value)

	return &uw
}

// Prove proves a key in the tree.
func (t *Tree) Prove(key chainhash.Hash) *VerificationWitness {
	oldVal := t.Get(key)
	t.Set(key, *oldVal)
	vw := GenerateVerificationWitness(t, key)
	return &vw
}

func isRight(key chainhash.Hash, level uint8) bool {
	return key[level/8]&(1<<uint(level%8)) != 0
}

// calculateSubtreeHashWithOneLeaf calculates the hash of a subtree with only a single leaf at a certain height.
// atLevel is the height to calculate at.
func calculateSubtreeHashWithOneLeaf(key *chainhash.Hash, value *chainhash.Hash, atLevel uint8) chainhash.Hash {
	h := *value

	for i := uint8(0); i < atLevel; i++ {
		right := isRight(*key, i+1)

		// the key is in the right subtree
		if right {
			h = combineHashes(&emptyTrees[i], &h)
		} else {
			h = combineHashes(&h, &emptyTrees[i])
		}
	}

	return h
}

func (t *Tree) insert(root *Node, key *chainhash.Hash, value *chainhash.Hash, level uint8) *Node {
	right := isRight(*key, level)

	if level == 0 {
		if root != nil {
			root.Value = *value
		}
		return &Node{
			Value: *value,
		}
	}

	// if this tree is empty and we're inserting, we know it's the only key in the subtree, so let's mark it as such and
	// fill in the necessary values
	if root == nil {
		return &Node{
			One:      true,
			OneKey:   key,
			OneValue: value,
			Value:    calculateSubtreeHashWithOneLeaf(key, value, level),
		}
	}

	// if there is only one key in this subtree,
	if root.One {
		// this operation is an update
		if root.OneKey.IsEqual(key) {
			root.Value = calculateSubtreeHashWithOneLeaf(key, value, level)
			root.OneValue = value
			return root
		}
		// we also need to add the old key to a lower sub-level.
		subRight := isRight(*root.OneKey, level)
		if subRight {
			root.Right = t.insert(root.Right, root.OneKey, root.OneValue, level-1)
		} else {
			root.Left = t.insert(root.Left, root.OneKey, root.OneValue, level-1)
		}
		root.One = false
		root.OneKey = nil
		root.OneValue = nil
	}

	if right {
		root.Right = t.insert(root.Right, key, value, level-1)
	} else {
		root.Left = t.insert(root.Left, key, value, level-1)
	}

	lv := emptyTrees[level-1]
	if root.Left != nil {
		lv = root.Left.Value
	}

	rv := emptyTrees[level-1]
	if root.Right != nil {
		rv = root.Right.Value
	}

	root.Value = combineHashes(&lv, &rv)

	return root
}

// Get gets a value from the tree.
func (t *Tree) Get(key chainhash.Hash) *chainhash.Hash {
	h, found := t.datastore[key]
	if !found {
		return &emptyHash
	}
	return &h
}

// Copy copies the tree.
func (t *Tree) Copy() Tree {
	newTree := Tree{
		datastore: map[chainhash.Hash]chainhash.Hash{},
	}

	for k, v := range t.datastore {
		newTree.datastore[k] = v
	}

	newTree.root = t.root.Copy()

	return newTree
}
