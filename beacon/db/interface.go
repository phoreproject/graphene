package db

import (
	"github.com/phoreproject/synapse/beacon/primitives"
	"github.com/phoreproject/synapse/chainhash"
)

// Database is a very basic interface for pluggable
// databases.
type Database interface {
	GetBlockForHash(h chainhash.Hash) (*primitives.Block, error)
	SetBlock(b primitives.Block) error
}
