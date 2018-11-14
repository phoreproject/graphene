package db

import (
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/inconshreveable/log15"
	"github.com/phoreproject/synapse/primitives"
)

// InMemoryDB is a very basic block database.
type InMemoryDB struct {
	DB   map[chainhash.Hash]primitives.Block
	lock *sync.Mutex
}

// NewInMemoryDB initializes a new in-memory DB
func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{DB: make(map[chainhash.Hash]primitives.Block), lock: &sync.Mutex{}}
}

// GetBlockForHash is a database lookup function
func (imdb *InMemoryDB) GetBlockForHash(h chainhash.Hash) (*primitives.Block, error) {
	imdb.lock.Lock()
	defer imdb.lock.Unlock()
	out, found := imdb.DB[h]
	log15.Debug("retrieving block from memory db", "hash", h, "success", found)
	if !found {
		return nil, fmt.Errorf("could not find block with hash")
	}
	return &out, nil
}

// SetBlock adds the block to storage
func (imdb *InMemoryDB) SetBlock(b primitives.Block) error {
	imdb.lock.Lock()
	log15.Debug("setting block in memory db", "block", b.Hash(), "slot", b.SlotNumber)
	imdb.DB[b.Hash()] = b
	imdb.lock.Unlock()
	return nil
}
