package rpc

import (
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/chain"
	"github.com/prysmaticlabs/go-ssz"
)

// ActionStreamGenerator listens for block actions and relays them to the callback.
type ActionStreamGenerator struct {
	cb func(action *pb.ShardChainAction)
}

// NewActionStreamGenerator creates a new action stream generator to use as a notifee to blockchain.
func NewActionStreamGenerator(cb func(action *pb.ShardChainAction)) *ActionStreamGenerator {
	return &ActionStreamGenerator{
		cb: cb,
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
func (a *ActionStreamGenerator) FinalizeBlockHash(blockHash chainhash.Hash) {
	a.cb(&pb.ShardChainAction{
		FinalizeBlockAction: &pb.ActionFinalizeBlock{
			Hash: blockHash[:],
		},
	})
}

var _ chain.ShardChainActionNotifee = &ActionStreamGenerator{}
