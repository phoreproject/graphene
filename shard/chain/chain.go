package chain

import (
	"errors"
	"fmt"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/go-ssz"
	"sync"
)

// ShardChain is a chain of shard block nodes that form a chain.
type ShardChain struct {
	lock     *sync.Mutex
	RootSlot uint64
	chain    []*ShardBlockNode
}

// NewShardChain creates a new shard chain.
func NewShardChain(rootSlot uint64, genesisBlock *primitives.ShardBlock) *ShardChain {
	genesisHash, err := ssz.HashTreeRoot(genesisBlock)
	if err != nil {
		panic(err)
	}

	return &ShardChain{
		lock:     new(sync.Mutex),
		RootSlot: rootSlot,
		chain: []*ShardBlockNode{
			{
				Parent:    nil,
				BlockHash: genesisHash,
				StateRoot: chainhash.Hash{},
				Slot:      0,
				Height:    0,
			},
		},
	}
}

// GetNodeBySlot gets the block node at a certain slot.
func (c *ShardChain) GetNodeBySlot(slot uint64) (*ShardBlockNode, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if slot < c.RootSlot {
		return nil, fmt.Errorf("do not have slot %d, earliest slot is: %d", slot, c.RootSlot)
	}

	tip, err := c.tip()
	if err != nil {
		return nil, err
	}

	if slot > tip.Slot {
		return nil, fmt.Errorf("do not have slot %d, tip slot is: %d", slot, tip.Slot)
	}

	return tip.GetAncestorAtSlot(slot), nil
}

// Tip gets the block node of the tip of the blockchain.
func (c *ShardChain) Tip() (*ShardBlockNode, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.tip()
}

// tip gets the block node of the tip. Should be called with the lock held.
func (c *ShardChain) tip() (*ShardBlockNode, error) {
	if len(c.chain) == 0 {
		return nil, errors.New("empty blockchain")
	}

	return c.chain[len(c.chain)-1], nil
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
