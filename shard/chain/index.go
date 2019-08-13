package chain

import "github.com/phoreproject/synapse/chainhash"

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

// NewShardBlockIndex creates a new block index.
func NewShardBlockIndex() *ShardBlockIndex {
	return &ShardBlockIndex{
		Index: make(map[chainhash.Hash]*ShardBlockNode),
	}
}
