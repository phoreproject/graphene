package blockchain

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/transaction"
)

// VoteCache is a cache of the total deposit of a committee
// and the validator indices in the committee.
type VoteCache struct {
	validatorIndices []uint32
	totalDeposit     uint64
}

// Copy makes a deep copy of the vote cache.
func (v *VoteCache) Copy() *VoteCache {
	voterIndices := make([]uint32, len(v.validatorIndices))
	copy(voterIndices, v.validatorIndices)

	return &VoteCache{
		validatorIndices: voterIndices,
		totalDeposit:     v.totalDeposit,
	}
}

// voteCacheDeepCopy copies the vote cache from a mapping of the
// blockhash to vote cache to a new mapping.
func voteCacheDeepCopy(old map[chainhash.Hash]*VoteCache) map[chainhash.Hash]*VoteCache {
	new := map[chainhash.Hash]*VoteCache{}
	for k, v := range old {
		newK := chainhash.Hash{}
		copy(newK[:], k[:])

		new[newK] = v.Copy()
	}

	return new
}

// CalculateNewVoteCache tallies votes for attestations in each block.
func (s *State) CalculateNewVoteCache(block *primitives.Block, cache map[chainhash.Hash]*VoteCache, c *Config) (map[chainhash.Hash]*VoteCache, error) {
	newCache := voteCacheDeepCopy(cache)

	for _, a := range block.Attestations {
		parentHashes, err := s.Active.getSignedParentHashes(block, &a, c)
		if err != nil {
			return nil, err
		}

		attesterIndices, err := s.Crystallized.GetAttesterIndices(&a, c)
		if err != nil {
			return nil, err
		}

		for _, h := range parentHashes {
			skip := false
			for _, o := range a.ObliqueParentHashes {
				if o.IsEqual(&h) {
					// skip if part of oblique parent hashes
					skip = true
				}
			}
			if skip {
				continue
			}

			if _, success := cache[h]; !success {
				newCache[h] = &VoteCache{}
			}

			for i, attester := range attesterIndices {
				if !hasVoted(a.AttesterBitField, i) {
					continue
				}

				attesterExists := false
				for _, indexInCache := range newCache[h].validatorIndices {
					if attester == indexInCache {
						attesterExists = true
					}
				}

				if !attesterExists {
					newCache[h].totalDeposit += s.Crystallized.Validators[attester].Balance
					newCache[h].validatorIndices = append(newCache[h].validatorIndices, attester)
				}
			}
		}
	}

	return newCache, nil
}

func (a *ActiveState) getSignedParentHashes(block *primitives.Block, att *transaction.Attestation, c *Config) ([]chainhash.Hash, error) {
	recentHashes := a.RecentBlockHashes
	obliqueParentHashes := att.ObliqueParentHashes
	earliestSlot := int(block.SlotNumber) - len(recentHashes)

	startIdx := int(att.Slot) - earliestSlot - int(c.CycleLength) + 1
	endIdx := startIdx - len(att.ObliqueParentHashes) + int(c.CycleLength)

	if startIdx < 0 || endIdx > len(recentHashes) || endIdx <= startIdx {
		return nil, fmt.Errorf("attempt to fetch recent blockhashes from %d to %d invalid", startIdx, endIdx)
	}

	hashes := make([]chainhash.Hash, 0, c.CycleLength)
	for i := startIdx; i < endIdx; i++ {
		hashes = append(hashes, recentHashes[i])
	}

	for i := 0; i < len(obliqueParentHashes); i++ {
		hashes = append(hashes, obliqueParentHashes[i])
	}

	return hashes, nil
}
