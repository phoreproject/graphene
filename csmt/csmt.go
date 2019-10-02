package csmt

import (
	"github.com/phoreproject/synapse/chainhash"
)


func combineHashes(left *chainhash.Hash, right *chainhash.Hash) chainhash.Hash {
	return chainhash.HashH(append(left[:], right[:]...))
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

// isRight checks if the key is in the left or right subtree at a certain level. Level 255 is the root level.
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

func insertIntoTree(t TreeDatabase, root Node, key chainhash.Hash, value chainhash.Hash, level uint8) Node {
	right := isRight(key, level)

	if level == 0 {
		if !root.Empty() {
			root.SetHash(value)
			return root
		}
		return t.NewNodeWithHash(value)
	}

	// if this tree is empty and we're inserting, we know it's the only key in the subtree, so let's mark it as such and
	// fill in the necessary values
	if root.Empty() {
		return t.NewSingleNode(key, value, calculateSubtreeHashWithOneLeaf(&key, &value, level))
	}

	// if there is only one key in this subtree,
	if root.IsSingle() {
		rootKey := root.GetSingleKey()
		// this operation is an update
		if rootKey.IsEqual(&key) {
			// calculate the new root hash for this subtree
			root.SetHash(calculateSubtreeHashWithOneLeaf(&key, &value, level))
			root.SetSingleValue(value)
			return root
		}

		// we also need to add the old key to a lower sub-level.
		newRoot := t.NewNode()

		// check if the old key goes in the left or right
		subRight := isRight(rootKey, level)
		if subRight {
			newRoot.SetRight(insertIntoTree(t, root.Right(), rootKey, root.GetSingleValue(), level-1))
		} else {
			newRoot.SetLeft(insertIntoTree(t, root.Left(), rootKey, root.GetSingleValue(), level-1))
		}

		root = newRoot
	}

	if right {
		root.SetRight(insertIntoTree(t, root.Right(), key, value, level-1))
	} else {
		root.SetLeft(insertIntoTree(t, root.Left(), key, value, level-1))
	}

	rootLeft := root.Left()
	lv := emptyTrees[level-1]
	if !rootLeft.Empty() {
		lv = rootLeft.GetHash()
	}

	rootRight := root.Right()
	rv := emptyTrees[level-1]
	if !rootRight.Empty() {
		rv = rootRight.GetHash()
	}

	root.SetHash(combineHashes(&lv, &rv))

	return root
}