package db

import (
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// ShardBlockNodeDisk is an on-disk representation of a ShardBlockNode.
type ShardBlockNodeDisk struct {
	ParentRoot chainhash.Hash
	BlockHash  chainhash.Hash
	StateRoot  chainhash.Hash
	Slot       uint64
	Height     uint64
}

// ShardBlockDatabase is an interface that represents access to the shard block node database.
type ShardBlockDatabase interface {
	GetBlockForHash(chainhash.Hash) (*primitives.ShardBlock, error)
	SetBlock(*primitives.ShardBlock) error
	SetBlockNode(*ShardBlockNodeDisk) error
	GetBlockNode(chainhash.Hash) (*ShardBlockNodeDisk, error)
	SetFinalizedHead(chainhash.Hash) error
	GetFinalizedHead() (*chainhash.Hash, error)
	SetChainTip(chainhash.Hash) error
	GetChainTip() (*chainhash.Hash, error)
	SetCodeForShardId(uint64, []byte) error
	GetCodeForShardId(uint64) ([]byte, error)
	Close() error
}
