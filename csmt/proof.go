package csmt

import (
	"fmt"
)

// directions
// For DirLeft, it means the proof hash goes left
const (
	DirLeft  = 1
	DirRight = 2
)

// Proof represents the proof interface
type Proof interface {
	IsMembershipProof() bool
}

// MembershipProofEntry is the entry in a MembershipProof
type MembershipProofEntry struct {
	hash      *Hash
	direction int
}

// GetHash gets hash
func (pe MembershipProofEntry) GetHash() *Hash {
	return pe.hash
}

// MembershipProof is a Proof for membership
type MembershipProof struct {
	node    LeafNode
	entries []*MembershipProofEntry
}

// NonMembershipProof is a Proof for non-membership
type NonMembershipProof struct {
	leftBoundProof  *MembershipProof
	rightBoundProof *MembershipProof
}

// IsMembershipProof implements Proof
func (p MembershipProof) IsMembershipProof() bool {
	return true
}

// IsMembershipProof implements Proof
func (p NonMembershipProof) IsMembershipProof() bool {
	return false
}

// GetEntryList returns the entries
func (p MembershipProof) GetEntryList() []*MembershipProofEntry {
	return p.entries
}

// SetEntryList sets the entries
func (p *MembershipProof) SetEntryList(entryList []*MembershipProofEntry) {
	p.entries = entryList
}

// ComputeRootHash computes root hash of the proof
func (p MembershipProof) ComputeRootHash(hash *Hash, calculator NodeHashFunction) *Hash {
	h := *hash
	for i := 0; i < len(p.entries); i++ {
		if p.entries[i].direction == DirLeft {
			h = calculator(p.entries[i].hash, &h)
		} else {
			h = calculator(&h, p.entries[i].hash)
		}
	}
	return &h
}

// VerifyHash verifies if a hash is valid
func (p MembershipProof) VerifyHash(hash *Hash, rootHash *Hash, calculator NodeHashFunction) bool {
	return p.ComputeRootHash(hash, calculator).IsEqual(rootHash)
}

// VerifyHashInTree verifies if a hash is valid in a CSMT
func (p MembershipProof) VerifyHashInTree(hash *Hash, tree *CSMT) bool {
	return p.VerifyHash(hash, tree.GetRootHash(), tree.nodeHashFunction)
}

// DebugToString returns a string, for debug purpose
func (p MembershipProof) DebugToString() string {
	result := ""
	for i, e := range p.entries {
		if i > 0 {
			result += ","
		}
		direction := "L"
		if e.direction == DirRight {
			direction = "R"
		}
		result += fmt.Sprintf("%s %s", direction, e.hash.String())
	}
	return result
}
