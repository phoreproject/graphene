package shard

import (
	"fmt"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
)

func nodeHashFunction(left *csmt.Hash, right *csmt.Hash) csmt.Hash {
	return chainhash.HashH(append(left[:], right[:]...))
}

const (
	witOpSet    = 1
	witOpDel    = 2
	witOpVerify = 3
)

// Witness keeps track of information needed to update a state root
type Witness struct {
	operation int
	key       chainhash.Hash
	proof     csmt.MembershipProof
}

func newWitness(operation int, key chainhash.Hash, proof csmt.Proof) (Witness, error) {
	if !proof.IsMembershipProof() {
		return Witness{}, fmt.Errorf("Invalid proof")
	}

	return Witness{
		operation: operation,
		key:       key,
		proof:     proof.(csmt.MembershipProof),
	}, nil
}

func (w *Witness) deepClone() *Witness {
	return &Witness{
		operation: w.operation,
		key:       w.key,
		proof:     w.proof,
	}
}

// StorageTree keeps track of a full Merkle tree
type StorageTree struct {
	merkleTree csmt.CSMT
}

// NewStorageTree creates a StorageTree
func NewStorageTree() *StorageTree {
	return &StorageTree{
		merkleTree: csmt.NewCSMT(nodeHashFunction),
	}
}

// Set sets a key within the tree
func (tree *StorageTree) Set(key, value chainhash.Hash) {
	tree.merkleTree.Remove(&key)
	tree.merkleTree.Insert(&key, &value)
}

// SetWithWitness sets a key within the tree and returns the Witness
func (tree *StorageTree) SetWithWitness(key, value chainhash.Hash) (Witness, error) {
	tree.merkleTree.Remove(&key)
	tree.merkleTree.Insert(&key, &value)
	return newWitness(witOpSet, key, tree.merkleTree.GetProof(&key))
}

// Get gets a value from the key
func (tree *StorageTree) Get(key chainhash.Hash) (chainhash.Hash, error) {
	v, err := tree.merkleTree.GetValue(&key)
	if err != nil {
		return chainhash.Hash{}, err
	}

	return *v.(*chainhash.Hash), nil
}

// Delete deletes a key
func (tree *StorageTree) Delete(key chainhash.Hash) error {
	return tree.merkleTree.Remove(&key)
}

// DeleteWithWitness deletes a key from the tree and returns the witness needed to delete the key given only the state root
func (tree *StorageTree) DeleteWithWitness(key chainhash.Hash) (Witness, error) {
	witness, err := tree.ProveKey(key)
	if err != nil {
		return witness, err
	}
	witness.operation = witOpDel
	return witness, tree.merkleTree.Remove(&key)
}

// ProveKey proves a key
func (tree *StorageTree) ProveKey(key chainhash.Hash) (Witness, error) {
	return newWitness(witOpVerify, key, tree.merkleTree.GetProof(&key))
}

// Hash returns the root key
func (tree *StorageTree) Hash() chainhash.Hash {
	return *tree.merkleTree.GetRootHash()
}

type partialNode struct {
	witness Witness
	key     chainhash.Hash
	value   chainhash.Hash
}

func (pn *partialNode) deepClone() *partialNode {
	return &partialNode{
		witness: *pn.witness.deepClone(),
		key:     pn.key,
		value:   pn.value,
	}
}

// PartialTree keeps track of updates to the tree and can calculate a state root if provided correct witnesses
type PartialTree struct {
	rootHash   chainhash.Hash
	witnesses  []Witness
	keyNodeMap map[chainhash.Hash]*partialNode
	nodeList   []*partialNode
}

// NewPartialTree creates a PartialTree
func NewPartialTree(rootHash chainhash.Hash, witnesses []Witness) PartialTree {
	ptree := PartialTree{
		rootHash:   rootHash,
		witnesses:  witnesses,
		keyNodeMap: map[chainhash.Hash]*partialNode{},
		nodeList:   []*partialNode{},
	}
	ptree.initialize()
	return ptree
}

func (ptree *PartialTree) initialize() {
	for _, w := range ptree.witnesses {
		ptree.keyNodeMap[w.key] = &partialNode{
			witness: w,
			key:     w.key,
		}
		ptree.nodeList = append(ptree.nodeList, &partialNode{
			witness: w,
			key:     w.key,
		})
	}
}

func (ptree *PartialTree) findNode(key chainhash.Hash, op int) *partialNode {
	for _, node := range ptree.nodeList {
		if node.witness.operation == op && node.key.IsEqual(&key) {
			return node
		}
	}

	return nil
}

// Set sets a key in the merkle tree or returns an error if the witnesses provided do not provide enough information to set the key.
func (ptree *PartialTree) Set(key, value chainhash.Hash) error {
	node := ptree.findNode(key, witOpSet)
	if node == nil {
		return fmt.Errorf("Key %s is not found", key.String())
	}

	node.value = value

	return nil
}

// Delete deletes a key from the merkle tree or returns an error if the witnesses provided do not provide enough information to delete the key.
func (ptree *PartialTree) Delete(key chainhash.Hash) error {
	node := ptree.findNode(key, witOpDel)
	if node == nil {
		return fmt.Errorf("Key %s is not found", key.String())
	}

	return nil
}

// Verify verifies a key in the merkle tree or return an error if the witnesses provided doesn't provide enough information to verify the key.
func (ptree *PartialTree) Verify(key chainhash.Hash) error {
	node := ptree.findNode(key, witOpVerify)
	if node == nil {
		return fmt.Errorf("Key %s is not found", key.String())
	}

	if node.witness.proof.VerifyHash(&key, &ptree.rootHash, nodeHashFunction) {
		return nil
	}

	return fmt.Errorf("Verify failed")
}

// Hash calculates new hash of the Merkle tree
func (ptree *PartialTree) Hash() (chainhash.Hash, error) {
	return ptree.computeHash()
}

// ease for debug and test
func (ptree *PartialTree) computeHash() (chainhash.Hash, error) {
	var partialNodeList []*partialNode
	for _, node := range ptree.nodeList {
		partialNodeList = append(partialNodeList, node.deepClone())
	}

	findNode := func(key *chainhash.Hash) *partialNode {
		for _, node := range partialNodeList {
			if node.witness.key.IsEqual(key) {
				return node
			}
		}
		return nil
	}

	for _, node := range partialNodeList {
		if node.witness.operation == witOpDel {
			entryList := node.witness.proof.GetEntryList()
			node.witness.key = *entryList[0].GetHash()
			entryList = entryList[1:]
			if len(entryList) > 0 {
				n := findNode(entryList[0].GetHash())
				if n != nil {
					if n.witness.operation == witOpDel {
						entryList = entryList[1:]
					}
				}
			}
			node.witness.proof.SetEntryList(entryList[:])
		}
	}

	for _, node := range partialNodeList {
		key := node.witness.key
		n := findNode(&key)
		if n != nil {
			if n.witness.operation == witOpDel {
				continue
			} else {
				node.witness.key = n.key
			}
		}
	}

	index := len(partialNodeList) - 1

	h := partialNodeList[index].witness.proof.ComputeRootHash(&partialNodeList[index].witness.key, nodeHashFunction)

	return *h, nil
}
