package db

import (
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

type ShardBlockDatabase interface {
	GetBlockForHash(chainhash.Hash) (*primitives.ShardBlock, error)
	SetBlock(*primitives.ShardBlock) error
	Close() error
}
