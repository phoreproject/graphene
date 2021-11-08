package csmt

import (
	"errors"

	"github.com/phoreproject/graphene/primitives"

	"github.com/phoreproject/graphene/chainhash"
)

// GenerateUpdateWitness generates a witness that allows calculation of a new state root.
func GenerateUpdateWitness(tree TreeDatabaseTransaction, key chainhash.Hash, value chainhash.Hash) (*primitives.UpdateWitness, error) {
	hk := chainhash.HashH(key[:])

	oldValue, err := tree.Get(key)
	if err != nil {
		oldValue = &chainhash.Hash{}
	}

	if oldValue == nil {
		oldValue = &chainhash.Hash{}
	}

	uw := &primitives.UpdateWitness{
		Key:      key,
		OldValue: *oldValue,
		NewValue: value,
	}

	current, err := tree.Root()
	if err != nil {
		return nil, err
	}

	if current == nil || current.Empty() {
		uw.Witnesses = make([]chainhash.Hash, 0)
		uw.WitnessBitfield = chainhash.Hash{}
		uw.LastLevel = 255
		return uw, nil
	}

	w := make([]chainhash.Hash, 0)

	// if current == nil, we know the subtree is empty, so we can break

	level := uint8(255)

	for current != nil && !current.Empty() && !current.IsSingle() {
		right := isRight(hk, level)

		if right {
			leftNodeHash := current.Left()
			if leftNodeHash != nil {
				w = append(w, *leftNodeHash)
				uw.WitnessBitfield[level/8] |= 1 << uint(level%8)
			}

			rightHash := current.Right()
			if rightHash == nil {
				current = nil
			} else {
				current, err = tree.GetNode(*rightHash)
				if err != nil {
					return nil, err
				}
			}
		} else if !right {
			rightNodeHash := current.Right()
			if rightNodeHash != nil {
				w = append(w, *rightNodeHash)
				uw.WitnessBitfield[level/8] |= 1 << uint(level%8)
			}

			leftHash := current.Left()
			if leftHash == nil {
				current = nil
			} else {
				current, err = tree.GetNode(*leftHash)
				if err != nil {
					return nil, err
				}
			}
		}
		level--
	}

	if current != nil && !current.Empty() {
		existingKey := current.GetSingleKey()
		if !existingKey.IsEqual(&hk) {
			existingValue := current.GetSingleValue()
			// go down until we find the place where they branch
			for isRight(existingKey, level) == isRight(hk, level) {
				level--
			}
			level--
			w = append(w, calculateSubtreeHashWithOneLeaf(&existingKey, &existingValue, level))
			uw.WitnessBitfield[(level+1)/8] |= 1 << uint((level+1)%8)
		}
	}

	uw.LastLevel = level

	for i := len(w)/2 - 1; i >= 0; i-- {
		opp := len(w) - 1 - i
		w[i], w[opp] = w[opp], w[i]
	}

	uw.Witnesses = w

	return uw, nil
}

// GenerateVerificationWitness generates a witness that allows verification of a key in the tree.
func GenerateVerificationWitness(tree TreeDatabaseTransaction, key chainhash.Hash) (*primitives.VerificationWitness, error) {
	hk := chainhash.HashH(key[:])

	val, err := tree.Get(key)
	if err != nil {
		return nil, err
	}
	if val == nil {
		val = &chainhash.Hash{}
	}

	vw := &primitives.VerificationWitness{
		Key:   key,
		Value: *val,
	}

	current, err := tree.Root()
	if err != nil {
		return nil, err
	}

	if current == nil || current.Empty() {
		vw.Witnesses = make([]chainhash.Hash, 0)
		vw.WitnessBitfield = chainhash.Hash{}
		vw.LastLevel = 255
		return vw, nil
	}

	w := make([]chainhash.Hash, 0)

	// if current == nil, we know the subtree is empty, so we can break

	level := uint8(255)

	// we recurse down the tree until we find a subtree with only one root
	for current != nil && !current.Empty() && !current.IsSingle() {
		right := isRight(hk, level)

		if right {
			leftNodeHash := current.Left()
			if leftNodeHash != nil {
				w = append(w, *leftNodeHash)
				vw.WitnessBitfield[level/8] |= 1 << uint(level%8)
			}

			rightHash := current.Right()
			if rightHash == nil {
				current = nil
			} else {
				current, err = tree.GetNode(*rightHash)
				if err != nil {
					return nil, err
				}
			}
		} else if !right {
			rightNodeHash := current.Right()
			if rightNodeHash != nil {
				w = append(w, *rightNodeHash)
				vw.WitnessBitfield[level/8] |= 1 << uint(level%8)
			}

			leftHash := current.Left()
			if leftHash == nil {
				current = nil
			} else {
				current, err = tree.GetNode(*leftHash)
				if err != nil {
					return nil, err
				}
			}
		}
		level--
	}

	if current != nil && !current.Empty() {
		existingKey := current.GetSingleKey()
		if !existingKey.IsEqual(&hk) {
			existingValue := current.GetSingleValue()
			// go down until we find the place where they branch
			for isRight(existingKey, level) == isRight(hk, level) {
				level--
			}
			level--
			w = append(w, calculateSubtreeHashWithOneLeaf(&existingKey, &existingValue, level))
			vw.WitnessBitfield[(level+1)/8] |= 1 << uint((level+1)%8)
		}
	}

	vw.LastLevel = level

	for i := len(w)/2 - 1; i >= 0; i-- {
		opp := len(w) - 1 - i
		w[i], w[opp] = w[opp], w[i]
	}

	vw.Witnesses = w

	return vw, nil
}

// CalculateRoot calculates the root of the tree with the given witness information.
func CalculateRoot(key chainhash.Hash, value chainhash.Hash, witnessBitfield chainhash.Hash, witnesses []chainhash.Hash, lastLevel uint8) (*chainhash.Hash, error) {
	hk := chainhash.HashH(key[:])
	h := calculateSubtreeHashWithOneLeaf(&hk, &value, lastLevel)

	currentWitness := 0

	for i := uint16(lastLevel) + 1; i <= 255; i++ {
		right := isRight(hk, uint8(i))

		hashToAdd := primitives.EmptyTrees[i-1]
		if witnessBitfield[i/8]&(1<<uint8(i%8)) != 0 {
			if currentWitness >= len(witnesses) {
				return nil, errors.New("not enough witnesses")
			}
			hashToAdd = witnesses[currentWitness]
			currentWitness++
		}

		if right {
			h = primitives.CombineHashes(&hashToAdd, &h)
		} else {
			h = primitives.CombineHashes(&h, &hashToAdd)
		}
	}

	return &h, nil
}

// ApplyWitness applies a witness to an old state root to generate a new state root.
func ApplyWitness(uw primitives.UpdateWitness, oldStateRoot chainhash.Hash) (*chainhash.Hash, error) {
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

// CheckWitness ensures the state root matches.
func CheckWitness(vw *primitives.VerificationWitness, oldStateRoot chainhash.Hash) bool {
	preRoot, err := CalculateRoot(vw.Key, vw.Value, vw.WitnessBitfield, vw.Witnesses, vw.LastLevel)
	if err != nil {
		return false
	}

	if !preRoot.IsEqual(&oldStateRoot) {
		return false
	}

	return true
}
