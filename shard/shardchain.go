package shard

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/beacon/config"
	beacondb "github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/chainhash"
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

// Blockchain represents a chain of shard blocks.
type Blockchain struct {
	config        *config.Config
	ctx           context.Context
	blockchainRPC pb.BlockchainRPCClient
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
	if beaconRefBlock.BlockHeader.SlotNumber > header.SlotNumber {
		return fmt.Errorf("slot of shard block must not be larger than beacon block")
	}

	beaconStateProto, err := b.blockchainRPC.GetState(b.ctx, &empty.Empty{})
	if err != nil {
		return err
	}

	beaconState, err := primitives.StateFromProto(beaconStateProto.State)
	if err != nil {
		return err
	}

	committeeIndices, err := beaconState.GetCommitteeIndices(header.SlotNumber, header.ShardID, b.config)
	if err != nil {
		return err
	}

	if len(header.AttesterBitfield) != (len(committeeIndices)+7)/8 {
		return fmt.Errorf("attestation has incorrect bitfield length")
	}

	// Spec: Let curblock_proposer_index = hash(state.randao_mix + bytes8(shard_id) + bytes8(slot)) % len(validators).
	// Let parent_proposer_index be the same value calculated for the parent block.
	// Make sure that the parent_proposer_index'th bit in the attester_bitfield is set to 1.
	parentSlotNumberIDBuffer := make([]byte, 8)
	parentShardIDBuffer := make([]byte, 8)
	binary.BigEndian.PutUint64(parentSlotNumberIDBuffer, parentBlock.SlotNumber)
	binary.BigEndian.PutUint64(parentShardIDBuffer, parentBlock.ShardID)
	concatenatedBuffers := append(beaconState.RandaoMix.CloneBytes(), append(parentShardIDBuffer, parentSlotNumberIDBuffer...)...)
	concatedBufferHash := chainhash.HashB(concatenatedBuffers)
	var a, c, m big.Int
	a.SetBytes(concatedBufferHash)
	c.SetUint64(uint64(len(committeeIndices)))
	m.Mod(&a, &c)
	parentProposerIndex := m.Uint64()
	if !HasBitSetAt(header.AttesterBitfield, uint32(parentProposerIndex)) {
		return fmt.Errorf("bit at parentProposerIndex is not set in AttesterBitfield")
	}

	return nil
}
