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
	chain    []*ShardBlockNode
}

// NewShardChain creates a new shard chain.
func NewShardChain(rootSlot uint64) *ShardChain {
	return &ShardChain{
		lock:     new(sync.Mutex),
		RootSlot: rootSlot,
		chain:    []*ShardBlockNode{},
	}
}

// GetBlockHashAtSlot gets the block hash of a certain slot.
func (c *ShardChain) GetBlockHashAtSlot(slot uint64) (*chainhash.Hash, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if slot < c.RootSlot || slot > uint64(len(c.chain))+c.RootSlot {
		return nil, fmt.Errorf("do not have slot %d", slot)
	}

	return &c.chain[slot-c.RootSlot].BlockHash, nil
}

// Tip gets the block hash of the tip of the blockchain.
func (c *ShardChain) Tip() (*chainhash.Hash, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.chain) == 0 {
		return nil, errors.New("empty blockchain")
	}

	return &c.chain[len(c.chain)-1].BlockHash, nil
}

// SetTip sets the tip of the chain.
func (c *ShardChain) SetTip(node *ShardBlockNode) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if node == nil {
		c.chain = make([]*ShardBlockNode, 0)
		return
	}

	needed := node.Height + 1

	// algorithm copied from btcd chainview
	if uint64(cap(c.chain)) < needed {
		newChain := make([]*ShardBlockNode, needed, 128+needed)
		copy(newChain, c.chain)
		c.chain = newChain
	} else {
		prevLen := uint64(len(c.chain))
		c.chain = c.chain[0:needed]
		for i := prevLen; i < needed; i++ {
			c.chain[i] = nil
		}
	}

	for node != nil && c.chain[node.Height] != node {
		c.chain[node.Height] = node
		node = node.Parent
	}
}
