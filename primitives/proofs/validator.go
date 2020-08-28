package proofs

import (
	"encoding/binary"
	"fmt"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/primitives"
)

// GetValidatorHash gets the merkle root for all currently registered validators.
func GetValidatorHash(s *primitives.State, slotsPerEpoch uint64) chainhash.Hash {
	committeesPerShard := make(map[uint64]primitives.ShardAndCommittee)

	for _, slotAssignment := range s.ShardAndCommitteeForSlots[:slotsPerEpoch] {
		for _, committee := range slotAssignment {
			committeesPerShard[committee.Shard] = committee
		}
	}

	validatorMsgs := make(map[chainhash.Hash][32]byte, 0)
	for shardID, committee := range committeesPerShard {
		for idx, valIdx := range committee.Committee {
			val := s.ValidatorRegistry[valIdx]

			var key [16]byte
			binary.BigEndian.PutUint64(key[:], shardID)
			binary.BigEndian.PutUint64(key[8:], uint64(idx))

			k := chainhash.HashH(key[:])

			validatorMsgs[k] = chainhash.HashH(val.Pubkey[:])
		}
	}

	t := csmt.NewTree(csmt.NewInMemoryTreeDB())
	_ = t.Update(func(tx csmt.TreeTransactionAccess) error {
		for key, pkh := range validatorMsgs {
			if err := tx.Set(key, pkh); err != nil {
				return err
			}
		}
		return nil
	})

	h, _ := t.Hash()

	return h
}

// ConstructValidatorProof constructs a proof that a certain validator was in a
// certain committee and can be verified using the hash calculated in GetValidatorHash.
func ConstructValidatorProof(s *primitives.State, forValidator uint32, slotsPerEpoch uint64) (*primitives.ValidatorProof, error) {
	committeesPerShard := make(map[uint64]primitives.ShardAndCommittee)

	for _, slotAssignment := range s.ShardAndCommitteeForSlots[:slotsPerEpoch] {
		for _, committee := range slotAssignment {
			committeesPerShard[committee.Shard] = committee
		}
	}

	validatorKey := new(chainhash.Hash)
	proof := new(primitives.ValidatorProof)

	validatorMsgs := make(map[chainhash.Hash][32]byte, 0)
	for shardID, committee := range committeesPerShard {
		for idx, valIdx := range committee.Committee {
			val := s.ValidatorRegistry[valIdx]

			var key [16]byte
			binary.BigEndian.PutUint64(key[:], shardID)
			binary.BigEndian.PutUint64(key[8:], uint64(idx))

			k := chainhash.HashH(key[:])

			if valIdx == forValidator {
				validatorKey = &k
				proof.ShardID = shardID
				proof.PublicKey = val.Pubkey
				proof.ValidatorIndex = uint64(idx)
			}

			validatorMsgs[k] = chainhash.HashH(val.Pubkey[:])
		}
	}

	if validatorKey == nil {
		return nil, fmt.Errorf("validator %d not assigned", forValidator)
	}

	t := csmt.NewTree(csmt.NewInMemoryTreeDB())
	err := t.Update(func(tx csmt.TreeTransactionAccess) error {
		for key, pkh := range validatorMsgs {
			if err := tx.Set(key, pkh); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var vw *primitives.VerificationWitness

	err = t.View(func(txA csmt.TreeTransactionAccess) error {
		tx := txA.(*csmt.TreeTransaction)
		vw, err = tx.Prove(*validatorKey)
		return err
	})
	if err != nil {
		return nil, err
	}

	proof.Proof = *vw

	return proof, nil
}
