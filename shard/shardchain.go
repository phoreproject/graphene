package shard

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	beacon "github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/db"
	beacondb "github.com/phoreproject/synapse/beacon/db"
)

// BlockHeader represents a single shard chain block header.
type BlockHeader struct {
	// Slot number
	SlotNumber uint64
	// What shard is it on
	ShardID uint64
	// Parent block hash
	ParentHash chainhash.Hash
	// Beacon chain block
	BeaconChainRef chainhash.Hash
	// Depth of the Merkle tree
	DataTreeDepth uint8
	// Merkle root of data
	DataRoot chainhash.Hash
	// State root (placeholder for now)
	StateRoot chainhash.Hash
	// Attestation (including block signature)
	AttesterBitfield []uint8
	//AggregateSig: ['uint256']
	AggregateSig [][]uint8
}

// BlockBody represents the block body
type BlockBody struct {
}

// Block represents a single shard chain block
type Block struct {
	header BlockHeader
	body   BlockBody
}

type blockchainView struct {
	chain []chainhash.Hash
	lock  *sync.Mutex
}

// Blockchain represents a chain of shard blocks.
type Blockchain struct {
	chain  blockchainView
	db     db.Database
	config *beacon.Config
}

func (b *Blockchain) verifyBlockHeader(header *BlockHeader, shardDB Database, beaconDB beacondb.Database) error {
	parentBlock, err := shardDB.GetBlockForHash(header.ParentHash)
	if err != nil {
		return err
	}

	beaconRefBlock, err := beaconDB.GetBlockForHash(header.BeaconChainRef)
	if err != nil {
		return err
	}
	if beaconRefBlock.SlotNumber > header.SlotNumber {
		return fmt.Errorf("Slot of shard block must not be larger than beacon block")
	}

	// TODO: get the beaconState from beaconRefBlock
	var beaconState *beacon.BeaconState
	committeeIndices, err := beaconState.GetCommitteeIndices(header.SlotNumber, header.ShardID, b.config)
	if err != nil {
		return err
	}

	if len(header.AttesterBitfield) != (len(committeeIndices)+7)/8 {
		return fmt.Errorf("Attestation has incorrect bitfield length")
	}

	// Spec: Let curblock_proposer_index = hash(state.randao_mix + bytes8(shard_id) + bytes8(slot)) % len(validators).
	// Let parent_proposer_index be the same value calculated for the parent block.
	// Make sure that the parent_proposer_index'th bit in the attester_bitfield is set to 1.
	parentSlotNumberIDBuffer := make([]byte, 8)
	parentShardIDBuffer := make([]byte, 8)
	binary.BigEndian.PutUint64(parentSlotNumberIDBuffer, parentBlock.SlotNumber)
	binary.BigEndian.PutUint64(parentShardIDBuffer, parentBlock.ShardID)
	concatedBuffer := append(beaconState.RandaoMix.CloneBytes(), append(parentShardIDBuffer, parentSlotNumberIDBuffer...)...)
	concatedBufferHash := chainhash.HashB(concatedBuffer)
	var a, c, m big.Int
	a.SetBytes(concatedBufferHash)
	c.SetUint64(uint64(len(committeeIndices)))
	m.Mod(&a, &c)
	parentProposerIndex := m.Uint64()
	if !HasBitSetAt(header.AttesterBitfield, uint32(parentProposerIndex)) {
		return fmt.Errorf("Bit at parentProposerIndex is not set in AttesterBitfield")
	}

	return nil
}
