package db

import (
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// InMemoryDB is a very basic block database.
type InMemoryDB struct {
	DB   map[chainhash.Hash]*primitives.Block
	lock *sync.Mutex
}

// NewInMemoryDB initializes a new in-memory DB
func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{DB: make(map[chainhash.Hash]*primitives.Block), lock: &sync.Mutex{}}
}

// GetBlockForHash is a database lookup function
func (imdb *InMemoryDB) GetBlockForHash(h chainhash.Hash) (*primitives.Block, error) {
	imdb.lock.Lock()
	defer imdb.lock.Unlock()
	return imdb.DB[h], nil
}

// SetBlock adds the block to storage
func (imdb *InMemoryDB) SetBlock(b *primitives.Block) error {
	imdb.lock.Lock()
	imdb.DB[b.Hash()] = b
	imdb.lock.Unlock()
	return nil
}
