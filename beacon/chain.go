package beacon

import (
	"errors"
	"sync"

	"github.com/phoreproject/graphene/beacon/db"
	"github.com/phoreproject/graphene/chainhash"
	"github.com/phoreproject/graphene/primitives"
)

// BlockNode is an in-memory representation of a block.
type BlockNode struct {
	Hash          chainhash.Hash
	RANDAO        chainhash.Hash
	ValidatorRoot chainhash.Hash
	Height        uint64
	Slot          uint64
	Parent        *BlockNode
	StateRoot     chainhash.Hash
	Children      []*BlockNode
}

// GetAncestorAtHeight gets the ancestor of a block at a certain height.
func (node *BlockNode) GetAncestorAtHeight(height uint64) *BlockNode {
	if node.Height < height {
		return nil
	}

	current := node

	// go up to the slot after the slot we're searching for
	for height < current.Height {
		current = current.Parent
	}
	return current
}

// GetAncestorAtSlot gets the ancestor of a block at a certain slot.
// Defined if node.Slot >= slot.
func (node *BlockNode) GetAncestorAtSlot(slot uint64) *BlockNode {
	if node.Slot < slot {
		return nil
	}

	current := node

	// go up to the slot after the slot we're searching for
	for slot < current.Slot {
		current = current.Parent
	}
	return current
}

type blockNodeAndState struct {
	*BlockNode
	primitives.State
}

// BlockchainView is the state of GHOST-LMD
type BlockchainView struct {
	Chain *Chain
	Index *BlockIndex

	// lock protects the fields below it
	lock          *sync.Mutex
	finalizedHead blockNodeAndState
	justifiedHead blockNodeAndState
}

// NewBlockchainView creates a new blockchain view.
func NewBlockchainView() *BlockchainView {
	return &BlockchainView{
		Chain: NewChain(),
		Index: NewBlockIndex(),
		lock:  new(sync.Mutex),
	}
}

// SetFinalizedHead sets the finalized head of the blockchain.
func (bv *BlockchainView) SetFinalizedHead(finalizedHash chainhash.Hash, finalizedState primitives.State) bool {
	bv.lock.Lock()
	defer bv.lock.Unlock()
	finalizedNode := bv.Index.GetBlockNodeByHash(finalizedHash)
	if finalizedNode == nil {
		return false
	}
	bv.finalizedHead = blockNodeAndState{finalizedNode, finalizedState}
	return true
}

// SetJustifiedHead sets the justified head of the blockchain.
func (bv *BlockchainView) SetJustifiedHead(justifiedHash chainhash.Hash, justifiedState primitives.State) bool {
	bv.lock.Lock()
	defer bv.lock.Unlock()
	justifiedNode := bv.Index.GetBlockNodeByHash(justifiedHash)
	if justifiedNode == nil {
		return false
	}
	bv.justifiedHead = blockNodeAndState{justifiedNode, justifiedState}
	return true
}

// GetJustifiedHead gets the justified head of the blockchain.
func (bv *BlockchainView) GetJustifiedHead() (*BlockNode, primitives.State) {
	bv.lock.Lock()
	defer bv.lock.Unlock()

	return bv.justifiedHead.BlockNode, bv.justifiedHead.State
}

// GetFinalizedHead gets the finalized head of the blockchain.
func (bv *BlockchainView) GetFinalizedHead() (*BlockNode, primitives.State) {
	bv.lock.Lock()
	defer bv.lock.Unlock()

	return bv.finalizedHead.BlockNode, bv.finalizedHead.State
}

// BlockIndex is an index from Hash to block node.
type BlockIndex struct {
	index map[chainhash.Hash]*BlockNode
	lock  *sync.RWMutex
}

// NewBlockIndex creates a new block index.
func NewBlockIndex() *BlockIndex {
	return &BlockIndex{
		index: make(map[chainhash.Hash]*BlockNode),
		lock:  new(sync.RWMutex),
	}
}

func (bi *BlockIndex) getBlockNodeByHash(hash chainhash.Hash) *BlockNode {
	if node, found := bi.index[hash]; found {
		return node
	}
	return nil
}

// GetBlockNodeByHash gets a block node by hash.
func (bi *BlockIndex) GetBlockNodeByHash(hash chainhash.Hash) *BlockNode {
	bi.lock.RLock()
	defer bi.lock.RUnlock()

	return bi.getBlockNodeByHash(hash)
}

// ErrorNoParent occurs when a block that is being inserted does not have
// a valid parent block.
var ErrNoParent = errors.New("could not find parent node for new block node")

// AddBlockNodeToIndex adds a new block ot the blockchain.
func (bi *BlockIndex) AddBlockNodeToIndex(block *primitives.Block, blockHash chainhash.Hash, stateRoot chainhash.Hash, validatorRoot chainhash.Hash) (*BlockNode, error) {
	bi.lock.Lock()
	defer bi.lock.Unlock()

	if node := bi.getBlockNodeByHash(blockHash); node != nil {
		return node, nil
	}

	parentRoot := block.BlockHeader.ParentRoot
	parentNode := bi.getBlockNodeByHash(parentRoot)
	if block.BlockHeader.SlotNumber == 0 {
		parentNode = nil
	} else if parentNode == nil {
		return nil, ErrNoParent
	}

	height := uint64(0)
	if parentNode != nil {
		height = parentNode.Height + 1
	}

	node := &BlockNode{
		Hash:          blockHash,
		Height:        height,
		Slot:          block.BlockHeader.SlotNumber,
		Parent:        parentNode,
		StateRoot:     stateRoot,
		Children:      []*BlockNode{},
		ValidatorRoot: validatorRoot,
	}

	bi.index[blockHash] = node

	if parentNode != nil {
		parentNode.Children = append(parentNode.Children, node)
	}

	return node, nil
}

// LoadBlockNode loads a block node from disk. The parent must have already been added.
func (bi *BlockIndex) LoadBlockNode(blockNodeDisk *db.BlockNodeDisk) (*BlockNode, error) {
	bi.lock.Lock()
	defer bi.lock.Unlock()
	parent := bi.getBlockNodeByHash(blockNodeDisk.Parent)
	if parent == nil && blockNodeDisk.Slot != 0 {
		return nil, ErrNoParent
	}

	newNode := &BlockNode{
		Hash:      blockNodeDisk.Hash,
		Height:    blockNodeDisk.Height,
		Slot:      blockNodeDisk.Slot,
		StateRoot: blockNodeDisk.StateRoot,
		Parent:    parent,
		Children:  make([]*BlockNode, 0),
	}

	bi.index[blockNodeDisk.Hash] = newNode

	if parent != nil {
		parent.Children = append(parent.Children, newNode)
	}

	return newNode, nil
}

func (bi *BlockIndex) has(h chainhash.Hash) bool {
	_, found := bi.index[h]
	return found
}

// Has returns true if the block index has the specified block.
func (bi *BlockIndex) Has(h chainhash.Hash) bool {
	bi.lock.Lock()
	defer bi.lock.Unlock()

	return bi.has(h)
}

// Chain is a representation of the current main chain.
type Chain struct {
	lock  *sync.Mutex
	chain []*BlockNode
}

// NewChain creates a new chain.
func NewChain() *Chain {
	return &Chain{
		lock:  new(sync.Mutex),
		chain: make([]*BlockNode, 0),
	}
}

// Genesis gets the genesis of the chain
func (c *Chain) Genesis() *BlockNode {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.chain) > 0 {
		return c.chain[0]
	}
	return nil
}

// Height returns the height of the chain.
func (c *Chain) Height() int64 {
	c.lock.Lock()
	defer c.lock.Unlock()

	return int64(len(c.chain) - 1)
}

func (c *Chain) contains(node *BlockNode) bool {
	return c.chain[node.Height] == node
}

// Contains checks if the chain contains a BlockNode.
func (c *Chain) Contains(node *BlockNode) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.contains(node)
}

// GetBlockByHeight gets a block at a certain height or
// if it doesn't exist, returns nil.
func (c *Chain) GetBlockByHeight(height int) *BlockNode {
	c.lock.Lock()
	defer c.lock.Unlock()

	if height < len(c.chain) && height >= 0 {
		return c.chain[height]
	}
	return nil
}

// Tip gets the tip of the chain. Returns nil if no genesis block defined.
func (c *Chain) Tip() *BlockNode {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.chain) == 0 {
		return nil
	}
	return c.chain[len(c.chain)-1]
}

// GetChainLocator gets a chain locator by requesting blocks at certain heights.
// This code is basically copied from the Bitcoin code.
func (c *Chain) GetChainLocator() [][]byte {
	step := uint64(1)
	locator := make([][]byte, 0, 32)

	current := c.Tip()

	for {
		locator = append(locator, current.Hash[:])

		if current.Height == 0 {
			break
		}

		nextHeight := int(current.Height) - int(step)
		if nextHeight < 0 {
			nextHeight = 0
		}

		nextCurrent := c.GetBlockByHeight(nextHeight)

		step *= 2
		current = nextCurrent
	}

	return locator
}

// SetTip sets the tip of the chain.
func (c *Chain) SetTip(node *BlockNode) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if node == nil {
		c.chain = make([]*BlockNode, 0)
		return
	}

	needed := node.Height + 1

	// algorithm copied from btcd chainview
	if uint64(cap(c.chain)) < needed {
		newChain := make([]*BlockNode, needed, 1000+needed)
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

func (c *Chain) next(node *BlockNode) *BlockNode {
	if c.contains(node) && uint64(len(c.chain)) > node.Height+1 {
		return c.chain[node.Height+1]
	}
	return nil
}

// Next gets the next node in the chain.
func (c *Chain) Next(node *BlockNode) *BlockNode {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.next(node)
}

// GetBlockBySlot gets the block node at a certain slot.
func (c *Chain) GetBlockBySlot(slot uint64) *BlockNode {
	tip := c.Tip()
	if tip == nil {
		return nil
	}
	if tip.Slot < slot {
		return tip
	}
	return tip.GetAncestorAtSlot(slot)
}
