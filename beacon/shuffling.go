package beacon

import (
	"encoding/binary"
	"math"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// GetActiveValidatorIndices gets the indices of active validators.
func GetActiveValidatorIndices(vs []primitives.Validator) []uint32 {
	var l []uint32
	for i, v := range vs {
		if v.Status == primitives.Active {
			l = append(l, uint32(i))
		}
	}
	return l
}

// ShuffleValidators shuffles an array of ints given a seed.
func ShuffleValidators(toShuffle []uint32, seed chainhash.Hash) []uint32 {
	shuffled := toShuffle[:]
	numValues := len(toShuffle)

	randBytes := 3
	randMax := uint32(math.Pow(2, float64(randBytes*8)) - 1)

	source := seed
	index := 0
	for index < numValues-1 {
		source = chainhash.HashH(source[:])
		for position := 0; position < (32 - (32 % randBytes)); position += randBytes {
			remaining := uint32(numValues - index)
			if remaining == 1 {
				break
			}

			sampleFromSource := binary.BigEndian.Uint32(append([]byte{'\x00'}, source[position:position+randBytes]...))

			sampleMax := randMax - randMax%remaining

			if sampleFromSource < sampleMax {
				replacementPos := (sampleFromSource % remaining) + uint32(index)
				shuffled[index], shuffled[replacementPos] = shuffled[replacementPos], shuffled[index]
				index++
			}
		}
	}
	return shuffled
}

// Split splits an array into N different sections.
func Split(l []uint32, splitCount uint32) [][]uint32 {
	out := make([][]uint32, splitCount)
	numItems := uint32(len(l))
	for i := uint32(0); i < splitCount; i++ {
		out[i] = l[(numItems * i / splitCount):(numItems * (i + 1) / splitCount)]
	}
	return out
}

func clamp(min int, max int, val int) int {
	if val <= min {
		return min
	} else if val >= max {
		return max
	} else {
		return val
	}
}

// GetNewShuffling calculates the new shuffling of validators
// to slots and shards.
func GetNewShuffling(seed chainhash.Hash, validators []primitives.Validator, crosslinkingStart int, con *config.Config) [][]primitives.ShardAndCommittee {
	activeValidators := GetActiveValidatorIndices(validators)
	numActiveValidators := len(activeValidators)

	// clamp between 1 and b.config.ShardCount / b.config.EpochLength
	committeesPerSlot := clamp(1, con.ShardCount/int(con.EpochLength), numActiveValidators/int(con.EpochLength)/con.TargetCommitteeSize)

	output := make([][]primitives.ShardAndCommittee, con.EpochLength)

	shuffledValidatorIndices := ShuffleValidators(activeValidators, seed)

	validatorsPerSlot := Split(shuffledValidatorIndices, uint32(con.EpochLength))

	for slot, slotIndices := range validatorsPerSlot {
		shardIndices := Split(slotIndices, uint32(committeesPerSlot))

		shardIDStart := crosslinkingStart + slot*committeesPerSlot

		shardCommittees := make([]primitives.ShardAndCommittee, len(shardIndices))
		for shardPosition, indices := range shardIndices {
			shardCommittees[shardPosition] = primitives.ShardAndCommittee{
				Shard:     uint64((shardIDStart + shardPosition) % con.ShardCount),
				Committee: indices,
			}
		}

		output[slot] = shardCommittees
	}
	return output
}
