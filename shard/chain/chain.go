package chain

import (
	"errors"
	"fmt"
	"github.com/phoreproject/synapse/chainhash"
	"sync"
)

// ShardChain is a chain of shard block nodes that form a chain.
type ShardChain struct {
	lock     *sync.Mutex
	RootSlot uint64
	Chain    []*ShardBlockNode
}

// NewShardChain creates a new shard chain.
func NewShardChain(rootSlot uint64) *ShardChain {
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

// Tip gets the block hash of the tip of the blockchain.
func (c *ShardChain) Tip() (*chainhash.Hash, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.Chain) == 0 {
		return nil, errors.New("empty blockchain")
	}

	return &c.Chain[len(c.Chain)-1].BlockHash, nil
}
