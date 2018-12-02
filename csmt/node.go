package csmt

// Node is the interface of a node
type Node interface {
	GetKey() *Key
	GetHash() *Hash
	IsLeaf() bool
}

type nodeBase struct {
	key  Key
	hash Hash
}

func (node nodeBase) GetKey() *Key {
	return &node.key
}

func (node nodeBase) GetHash() *Hash {
	return &node.hash
}

// LeafNode is the leaf node
type LeafNode struct {
	nodeBase
}

// IsLeaf implements Node
func (node LeafNode) IsLeaf() bool {
	return true
}

// InnerNode is the non-leaf node
type InnerNode struct {
	nodeBase
	left  Node
	right Node
}

// IsLeaf implements Node
func (node InnerNode) IsLeaf() bool {
	return false
}

// GetLeft returns the left child node
func (node InnerNode) GetLeft() Node {
	return node.left
}

// GetRight returns the right child node
func (node InnerNode) GetRight() Node {
	return node.right
}

// NewLeafNode constructs a new LeafNode
func NewLeafNode(key *Key, hash *Hash) LeafNode {
	return LeafNode{
		nodeBase: nodeBase{
			key:  *key,
			hash: *hash,
		},
	}
}

// NewInnerNode constructs a new InnerNode
func NewInnerNode(hash Hash, left Node, right Node) InnerNode {
	return InnerNode{
		nodeBase: nodeBase{
			key:  *getMaxKey(left.GetKey(), right.GetKey()),
			hash: hash,
		},
		left:  left,
		right: right,
	}
}
