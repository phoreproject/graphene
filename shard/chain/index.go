package chain

import (
	"fmt"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/go-ssz"
	"sync"
)

// ShardBlockNode is a block node in the shard chain.
type ShardBlockNode struct {
	Parent    *ShardBlockNode
	BlockHash chainhash.Hash
	StateRoot chainhash.Hash
	Slot      uint64
	Height    uint64
}

// ShardBlockIndex keeps a map of block hash to block.
type ShardBlockIndex struct {
	Lock  *sync.RWMutex
	Index map[chainhash.Hash]*ShardBlockNode
}

// AddToIndex adds a block to the block index.
func (i *ShardBlockIndex) AddToIndex(block primitives.ShardBlock) (*ShardBlockNode, error) {
	i.Lock.Lock()
	defer i.Lock.Unlock()
	parent, found := i.Index[block.Header.PreviousBlockHash]

	if !found {
		return nil, fmt.Errorf("missing parent block %s", block.Header.PreviousBlockHash)
	}

	blockHash, err := ssz.HashTreeRoot(block)
	if err != nil {
		return nil, err
	}

	node := &ShardBlockNode{
		Parent:    parent,
		BlockHash: blockHash,
		StateRoot: block.Header.StateRoot,
		Slot:      block.Header.Slot,
		Height:    parent.Height + 1,
	}

	i.Index[blockHash] = node

	return node, nil
}

// HasBlock returns true if the block index contains a certain block.
func (i *ShardBlockIndex) HasBlock(hash chainhash.Hash) bool {
	i.Lock.RLock()
	defer i.Lock.RUnlock()
	_, found := i.Index[hash]
	return found
}

// GetNodeByHash gets a block node by hash.
func (i *ShardBlockIndex) GetNodeByHash(h *chainhash.Hash) (*ShardBlockNode, error) {
	i.Lock.RLock()
	defer i.Lock.RUnlock()

	node, found := i.Index[*h]
	if !found {
		return nil, fmt.Errorf("do not have block with hash %s", h)
	}

	return node, nil
}

// NewShardBlockIndex creates a new block index.
func NewShardBlockIndex(genesisBlock primitives.ShardBlock) *ShardBlockIndex {
	index := &ShardBlockIndex{
		Index: make(map[chainhash.Hash]*ShardBlockNode),
		Lock:  new(sync.RWMutex),
	}

	genesisHash, err := ssz.HashTreeRoot(genesisBlock)
	if err != nil {
		panic(err)
	}

	index.Index[genesisHash] = &ShardBlockNode{
		Parent:    nil,
		BlockHash: genesisHash,
		StateRoot: genesisBlock.Header.StateRoot,
		Slot:      0,
		Height:    0,
	}

	return index
}
