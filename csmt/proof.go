package csmt

// directions
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

// VerifyHash verifies if a hash is valid
func (p MembershipProof) VerifyHash(hash *Hash, rootHash *Hash, calculator NodeHashFunction) bool {
	h := *hash
	for i := 0; i < len(p.entries); i++ {
		if p.entries[i].direction == DirLeft {
			h = calculator(p.entries[i].hash, &h)
		} else {
			h = calculator(&h, p.entries[i].hash)
		}
	}
	return h.IsEqual(rootHash)
}

// VerifyHashInTree verifies if a hash is valid in a CSMT
func (p MembershipProof) VerifyHashInTree(hash *Hash, tree *CSMT) bool {
	return p.VerifyHash(hash, tree.GetRootHash(), tree.nodeHashFunction)
}
