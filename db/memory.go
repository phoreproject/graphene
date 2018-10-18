package db

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// InMemoryDB is a very basic block database.
type InMemoryDB struct {
	DB map[chainhash.Hash]*primitives.Block
}

// NewInMemoryDB initializes a new in-memory DB
func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{DB: make(map[chainhash.Hash]*primitives.Block)}
}

// GetBlockForHash is a database lookup function
func (imdb *InMemoryDB) GetBlockForHash(h chainhash.Hash) (*primitives.Block, error) {
	return imdb.DB[h], nil
}

// SetBlock adds the block to storage
func (imdb *InMemoryDB) SetBlock(b *primitives.Block) error {
	imdb.DB[b.Hash()] = b
	return nil
}
