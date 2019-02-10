package shard

import (
	"github.com/phoreproject/synapse/chainhash"
)

// Database is a very basic interface for pluggable
// databases.
type Database interface {
	GetBlockForHash(h chainhash.Hash) (*BlockHeader, error)
}
