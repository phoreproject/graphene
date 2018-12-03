package beacon

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/beacon/primitives"
	"github.com/phoreproject/synapse/transaction"
)

// VoteCache is a cache of the total deposit of a committee
// and the validator indices in the committee that have already
// been counted.
type VoteCache struct {
	validatorIndices map[uint32]bool
	totalDeposit     uint64
}

// Copy makes a deep copy of the vote cache.
func (v *VoteCache) Copy() *VoteCache {
	voterIndices := make(map[uint32]bool)
	for k := range v.validatorIndices {
		voterIndices[k] = true
	}

	return &VoteCache{
		validatorIndices: voterIndices,
		totalDeposit:     v.totalDeposit,
	}
}

// NewVoteCache initializes a new vote cache.
func NewVoteCache() *VoteCache {
	return &VoteCache{
		validatorIndices: make(map[uint32]bool),
		totalDeposit:     0,
	}
}

// CalculateNewVoteCache tallies votes for attestations in each block.
func (s *BeaconState) CalculateNewVoteCache(block *primitives.Block, cache map[chainhash.Hash]*VoteCache, c *Config) error {
	for _, a := range block.Attestations {
		parentHashes, err := s.getSignedParentHashes(block, &a, c)
		if err != nil {
			return err
		}

		attesterIndices, err := s.GetAttesterIndices(a.Data.Slot, a.Data.Shard, c)
		if err != nil {
			return err
		}

		for _, h := range parentHashes {
			skip := false
			for _, o := range a.Data.ParentHashes {
				if o.IsEqual(&h) {
					// skip if part of oblique parent hashes
					skip = true
				}
			}
			if skip {
				continue
			}

			if _, success := cache[h]; !success {
				cache[h] = NewVoteCache()
			}

			for i, attester := range attesterIndices {
				if !hasVoted(a.AttesterBitfield, i) {
					continue
				}

				if _, found := cache[h].validatorIndices[attester]; found {
					continue
				}
				cache[h].totalDeposit += s.Validators[attester].Balance
				cache[h].validatorIndices[attester] = true
			}
		}
	}

	return nil
}

func (s *BeaconState) getSignedParentHashes(block *primitives.Block, att *transaction.AttestationRecord, c *Config) ([]chainhash.Hash, error) {
	recentHashes := s.RecentBlockHashes
	obliqueParentHashes := att.Data.ParentHashes
	earliestSlot := int(block.SlotNumber) - len(recentHashes)

	startIdx := int(att.Data.Slot) - earliestSlot - int(c.CycleLength) + 1
	endIdx := startIdx - len(att.Data.ParentHashes) + int(c.CycleLength)

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
