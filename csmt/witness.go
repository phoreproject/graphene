package csmt

import (
	"errors"

	"github.com/phoreproject/synapse/chainhash"
)

// UpdateWitness allows an executor to securely update the tree root so that only a single key is changed.
type UpdateWitness struct {
	Key             chainhash.Hash
	OldValue        chainhash.Hash
	NewValue        chainhash.Hash
	WitnessBitfield chainhash.Hash
	LastLevel       uint8
	Witnesses       []chainhash.Hash
}

// GenerateUpdateWitness generates a witness that allows calculation of a new state root.
func GenerateUpdateWitness(tree *Tree, key chainhash.Hash, value chainhash.Hash) UpdateWitness {
	hk := chainhash.HashH(key[:])

	oldValue := tree.Get(key)

	uw := UpdateWitness{
		Key:      key,
		OldValue: *oldValue,
		NewValue: value,
	}

	if tree.root == nil {
		uw.Witnesses = make([]chainhash.Hash, 0)
		uw.WitnessBitfield = chainhash.Hash{}
		uw.LastLevel = 255
		return uw
	}

	w := make([]chainhash.Hash, 0)

	current := tree.root

	// if current == nil, we know the subtree is empty, so we can break

	level := uint8(255)

	for current != nil && !current.One {
		right := isRight(hk, level)

		if right {
			if current.Left != nil {
				w = append(w, current.Left.Value)
				uw.WitnessBitfield[level/8] |= 1 << uint(level%8)
			}
			current = current.Right
		} else if !right {
			if current.Right != nil {
				w = append(w, current.Right.Value)
				uw.WitnessBitfield[level/8] |= 1 << uint(level%8)
			}
			current = current.Left
		}
		level--
	}

	if current != nil && !current.OneKey.IsEqual(&hk) {
		existingKey := *current.OneKey
		// go down until we find the place where they branch
		for isRight(existingKey, level) == isRight(hk, level) {
			level--
		}
		level--
		w = append(w, calculateSubtreeHashWithOneLeaf(current.OneKey, current.OneValue, level))
		uw.WitnessBitfield[(level+1)/8] |= 1 << uint((level+1)%8)
	}

	uw.LastLevel = level

	for i := len(w)/2 - 1; i >= 0; i-- {
		opp := len(w) - 1 - i
		w[i], w[opp] = w[opp], w[i]
	}

	uw.Witnesses = w

	return uw
}

// GenerateVerificationWitness generates a witness that allows verification of a key in the tree.
func GenerateVerificationWitness(tree *Tree, key chainhash.Hash) VerificationWitness {
	hk := chainhash.HashH(key[:])

	val := tree.Get(key)

	vw := VerificationWitness{
		Key:   key,
		Value: *val,
	}

	if tree.root == nil {
		vw.Witnesses = make([]chainhash.Hash, 0)
		vw.WitnessBitfield = chainhash.Hash{}
		vw.LastLevel = 255
		return vw
	}

	w := make([]chainhash.Hash, 0)

	current := tree.root

	// if current == nil, we know the subtree is empty, so we can break

	level := uint8(255)

	// we recurse down the tree until we find a subtree with only one root
	for current != nil && !current.One {
		right := isRight(hk, level)

		if right {
			if current.Left != nil {
				w = append(w, current.Left.Value)
				vw.WitnessBitfield[level/8] |= 1 << uint(level%8)
			}
			current = current.Right
		} else if !right {
			if current.Right != nil {
				w = append(w, current.Right.Value)
				vw.WitnessBitfield[level/8] |= 1 << uint(level%8)
			}
			current = current.Left
		}

		level--
	}

	if current != nil && !current.OneKey.IsEqual(&hk) {
		existingKey := *current.OneKey
		// go down until we find the place where they branch
		for isRight(existingKey, level) == isRight(hk, level) {
			level--
		}
		level--
		w = append(w, calculateSubtreeHashWithOneLeaf(current.OneKey, current.OneValue, level))
		vw.WitnessBitfield[(level+1)/8] |= 1 << uint((level+1)%8)
	}

	vw.LastLevel = level

	for i := len(w)/2 - 1; i >= 0; i-- {
		opp := len(w) - 1 - i
		w[i], w[opp] = w[opp], w[i]
	}

	vw.Witnesses = w

	return vw
}

// CalculateRoot calculates the root of the tree with the given witness information.
func CalculateRoot(key chainhash.Hash, value chainhash.Hash, witnessBitfield chainhash.Hash, witnesses []chainhash.Hash, lastLevel uint8) (*chainhash.Hash, error) {
	hk := chainhash.HashH(key[:])
	h := calculateSubtreeHashWithOneLeaf(&hk, &value, lastLevel)

	currentWitness := 0

	for i := uint16(lastLevel) + 1; i <= 255; i++ {
		right := isRight(hk, uint8(i))

		hashToAdd := emptyTrees[i-1]
		if witnessBitfield[i/8]&(1<<uint8(i%8)) != 0 {
			if currentWitness >= len(witnesses) {
				return nil, errors.New("not enough witnesses")
			}
			hashToAdd = witnesses[currentWitness]
			currentWitness++
		}

		if right {
			h = combineHashes(&hashToAdd, &h)
		} else {
			h = combineHashes(&h, &hashToAdd)
		}
	}

	return &h, nil
}

// Apply applies a witness to an old state root to generate a new state root.
func (uw *UpdateWitness) Apply(oldStateRoot chainhash.Hash) (*chainhash.Hash, error) {
	// if this is an update, last level should be the same for the pre root, but if this is an insertion, last level should
	// be one level higher

	preRoot, err := CalculateRoot(uw.Key, uw.OldValue, uw.WitnessBitfield, uw.Witnesses, uw.LastLevel)
	if err != nil {
		return nil, err
	}

	if !preRoot.IsEqual(&oldStateRoot) {
		return nil, errors.New("old state root doesn't match witness")
	}

	newRoot, err := CalculateRoot(uw.Key, uw.NewValue, uw.WitnessBitfield, uw.Witnesses, uw.LastLevel)
	if err != nil {
		return nil, err
	}

	return newRoot, nil
}

// Check ensures the state root matches.
func (vw *VerificationWitness) Check(oldStateRoot chainhash.Hash) bool {
	preRoot, err := CalculateRoot(vw.Key, vw.Value, vw.WitnessBitfield, vw.Witnesses, vw.LastLevel)
	if err != nil {
		return false
	}

	if !preRoot.IsEqual(&oldStateRoot) {
		return false
	}

	return true
}

// VerificationWitness allows an executor to verify a specific node in the tree.
type VerificationWitness struct {
	Key             chainhash.Hash
	Value           chainhash.Hash
	WitnessBitfield chainhash.Hash
	Witnesses       []chainhash.Hash
	LastLevel       uint8
}
