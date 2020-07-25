package chain

import (
	"errors"
	"fmt"
	"sync"

	"github.com/phoreproject/synapse/primitives"
	index "github.com/phoreproject/synapse/shard/chain/index"
	"github.com/prysmaticlabs/go-ssz"
)

// ShardChain is a chain of shard block nodes that form a chain.
type ShardChain struct {
	lock     *sync.Mutex
	RootSlot uint64
	chain    []*index.ShardBlockNode
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
		chain: []*index.ShardBlockNode{
			{
				Parent:    nil,
				BlockHash: genesisHash,
				StateRoot: genesisBlock.Header.StateRoot,
				Slot:      0,
				Height:    0,
			},
		},
	}
}

// GetNodeBySlot gets the block node at a certain slot.
func (c *ShardChain) GetNodeBySlot(slot uint64) (*index.ShardBlockNode, error) {
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

// GetLatestNodeBySlot gets the latest block node before or equal to a certain slot.
func (c *ShardChain) GetLatestNodeBySlot(slot uint64) (*index.ShardBlockNode, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if slot < c.RootSlot {
		return nil, fmt.Errorf("do not have slot %d, earliest slot is: %d", slot, c.RootSlot)
	}

	tip, err := c.tip()
	if err != nil {
		return nil, err
	}

	return tip.GetClosestAncestorAtSlot(slot), nil
}

// Tip gets the block node of the tip of the blockchain.
func (c *ShardChain) Tip() (*index.ShardBlockNode, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.tip()
}

// tip gets the block node of the tip. Should be called with the lock held.
func (c *ShardChain) tip() (*index.ShardBlockNode, error) {
	if len(c.chain) == 0 {
		return nil, errors.New("empty blockchain")
	}

	return c.chain[len(c.chain)-1], nil
}

// SetTip sets the tip of the chain.
func (c *ShardChain) SetTip(node *index.ShardBlockNode) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if node == nil {
		c.chain = make([]*index.ShardBlockNode, 0)
		return
	}

	needed := node.Height + 1

	// algorithm copied from btcd chainview
	if uint64(cap(c.chain)) < needed {
		newChain := make([]*index.ShardBlockNode, needed, 128+needed)
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

// Genesis gets the genesis shard node.
func (c *ShardChain) Genesis() *index.ShardBlockNode {
	return c.chain[0]
}

// Height gets the height of the current shard chain.
func (c *ShardChain) Height() uint64 {
	return uint64(len(c.chain))
}

// Contains checks if the chain contains a certain block node.
func (c *ShardChain) Contains(node *index.ShardBlockNode) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.contains(node)
}

func (c *ShardChain) contains(node *index.ShardBlockNode) bool {
	return c.chain[node.Height] == node
}

// Next gets the next block node in the chian.
func (c *ShardChain) Next(node *index.ShardBlockNode) *index.ShardBlockNode {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.contains(node) && uint64(len(c.chain)) > node.Height+1 {
		return c.chain[node.Height+1]
	}

	return nil
}

// GetBlockByHeight gets a block at a certain height or
// if it doesn't exist, returns nil.
func (c *ShardChain) GetBlockByHeight(height int) *index.ShardBlockNode {
	c.lock.Lock()
	defer c.lock.Unlock()

	if height < len(c.chain) {
		return c.chain[height]
	}
	return nil
}

// GetChainLocator gets a chain locator by requesting blocks at certain heights.
// This code is basically copied from the Bitcoin code.
func (c *ShardChain) GetChainLocator() ([][]byte, error) {
	step := uint64(1)
	locator := make([][]byte, 0, 32)

	current, err := c.Tip()
	if err != nil {
		return nil, err
	}

	for {
		locator = append(locator, current.BlockHash[:])

		if current.Height == 0 {
			break
		}

		nextHeight := int(current.Height) - int(step)
		if nextHeight < 0 {
			nextHeight = 0
		}

		nextCurrent := c.GetBlockByHeight(nextHeight)
		if nextCurrent == nil {
			panic("Assertion error: getChainLocator should never ask for block above current tip")
		}

		step *= 2
		current = nextCurrent
	}

	return locator, nil
}
