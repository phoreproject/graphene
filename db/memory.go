package db

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// InMemoryDB is a very basic block database.
type InMemoryDB struct {
	DB map[chainhash.Hash]primitives.Block
}

// NewInMemoryDB initializes a new in-memory DB
func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{DB: make(map[chainhash.Hash]primitives.Block)}
}

// GetBlockForHash is a database lookup function
func (imdb *InMemoryDB) GetBlockForHash(h chainhash.Hash) primitives.Block {
	return imdb.DB[h]
}

// SetBlock adds the block to storage
func (imdb *InMemoryDB) SetBlock(b primitives.Block) {
	imdb.DB[b.BlockHeader.Hash()] = b
}
