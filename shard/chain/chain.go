package chain

import (
	"fmt"
	"github.com/phoreproject/synapse/chainhash"
	"sync"
)

// ShardChainInitializationParameters are the initialization parameters from the crosslink.
type ShardChainInitializationParameters struct {
	RootBlockHash chainhash.Hash
	RootSlot      uint64
}

// ShardManager represents part of the blockchain on a specific shard (starting from a specific crosslinked block hash),
// and continued to a certain point.
type ShardManager struct {
	ShardID                  uint64
	Chain                    ShardChain
	Index                    ShardBlockIndex
	InitializationParameters ShardChainInitializationParameters
}

// ShardChain is a chain of shard block nodes that form a chain.
type ShardChain struct {
	lock     *sync.Mutex
	RootSlot uint64
	Chain    []*ShardBlockNode
}

// NewShardChain creates a new shard chain.
func NewShardChain(rootSlot uint64, rootBlockHash chainhash.Hash) *ShardChain {
	return &ShardChain{
		lock:     new(sync.Mutex),
		RootSlot: rootSlot,
		Chain:    []*ShardBlockNode{},
	}
}

// GetBlockHashAtSlot gets the block hash of a certain slot.
func (c *ShardChain) GetBlockHashAtSlot(slot uint64) (*chainhash.Hash, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if slot < c.RootSlot || slot > uint64(len(c.Chain))+c.RootSlot {
		return nil, fmt.Errorf("do not have slot %d", slot)
	}

	return &c.Chain[slot-c.RootSlot].BlockHash, nil
}

// ShardBlockNode is a block node in the shard chain.
type ShardBlockNode struct {
	Parent    *ShardBlockNode
	BlockHash chainhash.Hash
	StateRoot chainhash.Hash
	Slot      uint64
}

// ShardBlockIndex keeps a map of block hash to block.
type ShardBlockIndex struct {
	Index map[chainhash.Hash]*ShardBlockNode
}
