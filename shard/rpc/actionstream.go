package rpc

import (
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/chain"
	"github.com/phoreproject/synapse/ssz"
)

// ActionStreamGenerator listens for block actions and relays them to the callback.
type ActionStreamGenerator struct {
	cb func(action *pb.ShardChainAction)
	config *config.Config
}

// NewActionStreamGenerator creates a new action stream generator to use as a notifee to blockchain.
func NewActionStreamGenerator(cb func(action *pb.ShardChainAction), config *config.Config) *ActionStreamGenerator {
	return &ActionStreamGenerator{
		cb: cb,
		config: config,
	}
}

// AddBlock adds a block tot he blockchain.
func (a *ActionStreamGenerator) AddBlock(block *primitives.ShardBlock, newTip bool) {
	a.cb(&pb.ShardChainAction{
		AddBlockAction: &pb.ActionAddBlock{
			Block: block.ToProto(),
		},
	})
	blockHash, _ := ssz.HashTreeRoot(block)
	if newTip {
		a.cb(&pb.ShardChainAction{
			UpdateTip: &pb.ActionUpdateTip{
				Hash: blockHash[:],
			},
		})
	}
}

// FinalizeBlockHash is called when a block is finalized.
func (a *ActionStreamGenerator) FinalizeBlockHash(blockHash chainhash.Hash, slot uint64) {
	lastEpochSlot := slot - (slot % a.config.EpochLength)

	if lastEpochSlot >= a.config.EpochLength {
		lastEpochSlot = lastEpochSlot - a.config.EpochLength
	}
	a.cb(&pb.ShardChainAction{
		FinalizeBlockAction: &pb.ActionFinalizeBlock{
			Hash: blockHash[:],
			Slot: lastEpochSlot,
		},
	})
}

var _ chain.ShardChainActionNotifee = &ActionStreamGenerator{}
