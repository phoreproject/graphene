package csmt

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Compact Sparse Merkle Trees
// Paper: https://eprint.iacr.org/2018/955.pdf

// Key is the key type of a CSMT
type Key = chainhash.Hash

// Hash is the hash type of a CSMT
type Hash = Key

// SMT is Sparse Merkle Tree
type SMT interface {
	GetRootHash() *Hash
	Insert(leafHash *Hash) error
	GetProof(key *Key) Proof
}

// NodeHashFunction computes the hash for the children for a inner node
type NodeHashFunction func(*Hash, *Hash) Hash

// CSMT is Compact Sparse Merkle Tree
// It implements interface SMT
type CSMT struct {
	root             Node
	nodeHashFunction NodeHashFunction
}

// GetRootHash implements interface SMT
func (tree *CSMT) GetRootHash() *Hash {
	return tree.root.GetHash()
}

// Insert implements interface SMT
func (tree *CSMT) Insert(leafHash *Hash) error {
	if tree.root == nil {
		tree.root = tree.createLeafNode(leafHash)
	} else {
		node, err := tree.doInsert(tree.root, leafHash, leafHash)
		if err == nil {
			tree.root = node
		} else {
			return err
		}
	}
	return nil
}

func (tree *CSMT) createLeafNode(leafHash *Hash) Node {
	return NewLeafNode(leafHash, leafHash)
}

func (tree *CSMT) createInnerNode(left Node, right Node) Node {
	return NewInnerNode(tree.nodeHashFunction(left.GetHash(), right.GetHash()), left, right)
}

func (tree *CSMT) doInsert(node Node, key *Key, leafHash *Hash) (Node, error) {
	if node.IsLeaf() {
		if key.IsEqual(node.GetKey()) {
			return nil, fmt.Errorf("Key exists")
		}

		newLeaf := tree.createLeafNode(leafHash)
		if compareKey(key, node.GetKey()) < 0 {
			return tree.createInnerNode(newLeaf, node), nil
		}
		return tree.createInnerNode(node, newLeaf), nil
	}

	left := node.(InnerNode).GetLeft()
	right := node.(InnerNode).GetRight()

	leftDistance := distance(key, left.GetKey())
	rightDistance := distance(key, right.GetKey())

	if leftDistance == rightDistance {
		newLeaf := tree.createLeafNode(leafHash)
		minKey := getMinKey(left.GetKey(), right.GetKey())

		if compareKey(key, minKey) < 0 {
			return tree.createInnerNode(newLeaf, node), nil
		}
		return tree.createInnerNode(node, newLeaf), nil
	}

	if leftDistance < rightDistance {
		newNode, err := tree.doInsert(left, key, leafHash)
		if err != nil {
			return nil, err
		}
		return tree.createInnerNode(newNode, right), nil
	}

	newNode, err := tree.doInsert(right, key, leafHash)
	if err != nil {
		return nil, err
	}
	return tree.createInnerNode(left, newNode), nil
}

// GetProof gets the proof for key
func (tree *CSMT) GetProof(key *Key) Proof {
	if tree.root == nil {
		return NonMembershipProof{
			leftBoundProof:  nil,
			rightBoundProof: nil,
		}
	}

	if tree.root.IsLeaf() {
		rootProof := MembershipProof{
			node:    tree.root.(LeafNode),
			entries: []*MembershipProofEntry{},
		}

		if key.IsEqual(tree.root.GetKey()) {
			return rootProof
		}

		if compareKey(key, tree.root.GetKey()) < 0 {
			return NonMembershipProof{
				leftBoundProof:  nil,
				rightBoundProof: &rootProof,
			}
		}
		return NonMembershipProof{
			leftBoundProof:  &rootProof,
			rightBoundProof: nil,
		}
	}

	castedRoot := tree.root.(InnerNode)
	leftBound, rightBound := tree.findBounds(castedRoot, key)

	if leftBound != nil && leftBound.IsEqual(rightBound) {
		return *tree.findProof(castedRoot, leftBound)
	}

	if leftBound != nil && rightBound != nil {
		return NonMembershipProof{
			leftBoundProof:  tree.findProof(castedRoot, leftBound),
			rightBoundProof: tree.findProof(castedRoot, rightBound),
		}
	}

	if leftBound == nil {
		return NonMembershipProof{
			leftBoundProof:  nil,
			rightBoundProof: tree.findProof(castedRoot, rightBound),
		}
	}

	return NonMembershipProof{
		leftBoundProof:  tree.findProof(castedRoot, leftBound),
		rightBoundProof: nil,
	}
}

func (tree *CSMT) findProof(root InnerNode, key *Key) *MembershipProof {
	left := root.GetLeft()
	right := root.GetRight()

	leftDistance := distance(key, left.GetKey())
	rightDistance := distance(key, right.GetKey())

	var resultEntries []*MembershipProofEntry
	var resultNode LeafNode

	if leftDistance < rightDistance {
		resultEntries, resultNode = tree.findProofHelper(right, DirLeft, left, key)
	} else {
		resultEntries, resultNode = tree.findProofHelper(left, DirRight, right, key)
	}

	return &MembershipProof{
		node:    resultNode,
		entries: resultEntries,
	}
}

func (tree *CSMT) findProofHelper(sibling Node, direction int, node Node, key *Key) ([]*MembershipProofEntry, LeafNode) {
	if node.IsLeaf() {
		return []*MembershipProofEntry{
			&MembershipProofEntry{
				hash:      sibling.GetHash(),
				direction: reverseDirection(direction),
			},
		}, node.(LeafNode)
	}

	left := node.(InnerNode).GetLeft()
	right := node.(InnerNode).GetRight()

	leftDistance := distance(key, left.GetKey())
	rightDistance := distance(key, right.GetKey())

	var resultEntries []*MembershipProofEntry
	var resultNode LeafNode

	if leftDistance < rightDistance {
		resultEntries, resultNode = tree.findProofHelper(right, DirLeft, left, key)
	} else {
		resultEntries, resultNode = tree.findProofHelper(left, DirRight, right, key)
	}

	resultEntries = append(resultEntries, &MembershipProofEntry{hash: sibling.GetHash(), direction: reverseDirection(direction)})

	return resultEntries, resultNode
}

func (tree *CSMT) findBounds(root InnerNode, key *Key) (*Key, *Key) {
	left := root.GetLeft()
	right := root.GetRight()

	leftDistance := distance(key, left.GetKey())
	rightDistance := distance(key, right.GetKey())

	if leftDistance == rightDistance {
		if compareKey(key, root.GetKey()) > 0 {
			return right.GetKey(), nil
		}
		return nil, left.GetKey()
	}

	if leftDistance < rightDistance {
		return tree.findBoundsBySibling(right, DirLeft, left, key)
	}
	return tree.findBoundsBySibling(left, DirRight, right, key)
}

func (tree *CSMT) findBoundsBySibling(sibling Node, direction int, node Node, key *Key) (*Key, *Key) {
	if node.IsLeaf() {
		if key.IsEqual(node.GetKey()) {
			return key, key
		}

		return tree.findBoundsHelper(key, node, direction, sibling)
	}

	left := node.(InnerNode).GetLeft()
	right := node.(InnerNode).GetRight()

	leftDistance := distance(key, left.GetKey())
	rightDistance := distance(key, right.GetKey())

	if leftDistance == rightDistance {
		return tree.findBoundsHelper(key, node, direction, sibling)
	}

	var leftBound, rightBound *Key

	if leftDistance < rightDistance {
		leftBound, rightBound = tree.findBoundsBySibling(right, DirLeft, left, key)
	} else {
		leftBound, rightBound = tree.findBoundsBySibling(left, DirRight, right, key)
	}

	if rightBound == nil && direction == DirLeft {
		return leftBound, minInSubtree(sibling)
	}
	if leftBound == nil && direction == DirRight {
		return maxInSubtree(sibling), rightBound
	}

	return leftBound, rightBound
}

func (tree *CSMT) findBoundsHelper(key *Key, node Node, direction int, sibling Node) (*Key, *Key) {
	relation := compareKey(key, node.GetKey())
	if relation > 0 && direction == DirLeft {
		return node.GetKey(), minInSubtree(sibling)
	}
	if relation > 0 && direction == DirRight {
		return node.GetKey(), nil
	}
	if relation <= 0 && direction == DirLeft {
		return nil, minInSubtree(node)
	}
	return maxInSubtree(sibling), minInSubtree(node)
}

func maxInSubtree(node Node) *Key {
	return node.GetKey()
}

func minInSubtree(node Node) *Key {
	if node.IsLeaf() {
		return node.GetKey()
	}

	return minInSubtree(node.(InnerNode).GetLeft())
}

func getMaxKey(keyA *Key, keyB *Key) *Key {
	for i := 0; i < chainhash.HashSize; i++ {
		a := keyA[i]
		b := keyB[i]

		if a > b {
			return keyA
		}
		if a < b {
			return keyB
		}
	}

	return keyA
}

func getMinKey(keyA *Key, keyB *Key) *Key {
	for i := 0; i < chainhash.HashSize; i++ {
		a := keyA[i]
		b := keyB[i]

		if a < b {
			return keyA
		}
		if a > b {
			return keyB
		}
	}

	return keyA
}

func xorKey(keyA *Key, keyB *Key) Key {
	var result Key
	for i := 0; i < chainhash.HashSize; i++ {
		result[i] = keyA[i] ^ keyB[i]
	}
	return result
}

func compareKey(keyA *Key, keyB *Key) int {
	for i := 0; i < chainhash.HashSize; i++ {
		a := int(keyA[i])
		b := int(keyB[i])
		if a != b {
			return a - b
		}
	}

	return 0
}

// Fast calc log2(keyA ^ keyB)
func distance(keyA *Key, keyB *Key) int {
	result := chainhash.HashSize * 8
	for i := 0; i < chainhash.HashSize; i++ {
		b := keyA[i] ^ keyB[i]
		if b != 0 {
			var a uint8 = 0x80
			for k := 0; k < 8; k++ {
				if b&a != 0 {
					break
				}
				a >>= 1
				result--
			}
			break
		}

		result -= 8
	}
	return result
}

func reverseDirection(direction int) int {
	if direction == DirLeft {
		return DirRight
	}

	return DirLeft
}
