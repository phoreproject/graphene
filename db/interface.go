package db

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// Database is a very basic interface for pluggable
// databases.
type Database interface {
	GetBlockForHash(h chainhash.Hash) primitives.Block
	SetBlock(b primitives.Block)
}
