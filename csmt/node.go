package csmt

import (
	"fmt"
)

// Node is the interface of a node
type Node interface {
	GetKey() *Key
	GetHash() *Hash
	IsLeaf() bool
	DebugToJSONString() string
}

// For leaf node, key == hash
// For inner node, key is used for internal usage, it's the max key of the left and right child
// hash is the combined hash of left and right hash
type nodeBase struct {
	key  Key
	hash Hash
}

// GetKey gets the key of the node.
func (node nodeBase) GetKey() *Key {
	return &node.key
}

// GetHash gets the hash of the node.
func (node nodeBase) GetHash() *Hash {
	return &node.hash
}

// LeafNode is the leaf node
type LeafNode struct {
	nodeBase

	value interface{}
}

// NewLeafNode constructs a new LeafNode
func NewLeafNode(hash *Hash, value interface{}) LeafNode {
	return LeafNode{
		nodeBase: nodeBase{
			key:  *hash,
			hash: *hash,
		},

		value: value,
	}
}

// IsLeaf implements Node
func (node LeafNode) IsLeaf() bool {
	return true
}

// DebugToJSONString implements Node
func (node LeafNode) DebugToJSONString() string {
	result := ""
	result += "{"
	result += fmt.Sprintf("\"key\":\"%s\", ", node.GetKey().String())
	result += fmt.Sprintf("\"hash\":\"%s\" ", node.GetHash().String())
	result += "}"
	return result
}

// GetValue returns the value
func (node LeafNode) GetValue() interface{} {
	return node.value
}

// InnerNode is the non-leaf node
type InnerNode struct {
	nodeBase
	left  Node
	right Node
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

// DebugToJSONString implements Node
func (node InnerNode) DebugToJSONString() string {
	result := ""
	result += "{"
	result += fmt.Sprintf("\"key\":\"%s\", ", node.GetKey().String())
	result += fmt.Sprintf("\"hash\":\"%s\", ", node.GetHash().String())
	leftString := "null"
	if node.left != nil {
		leftString = node.left.DebugToJSONString()
	}
	rightString := "null"
	if node.right != nil {
		rightString = node.right.DebugToJSONString()
	}
	result += fmt.Sprintf("\"left\": %s, ", leftString)
	result += fmt.Sprintf("\"right\": %s ", rightString)
	result += "}"
	return result
}
