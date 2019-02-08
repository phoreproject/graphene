package db

import (
	"fmt"
	"sync"

	"github.com/phoreproject/prysm/shared/ssz"

	"github.com/phoreproject/synapse/beacon/primitives"
	"github.com/phoreproject/synapse/chainhash"
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
	if !found {
		return nil, fmt.Errorf("could not find block with hash")
	}
	return &out, nil
}

// SetBlock adds the block to storage
func (imdb *InMemoryDB) SetBlock(b primitives.Block) error {
	imdb.lock.Lock()
	blockHash, err := ssz.TreeHash(b)
	if err != nil {
		return err
	}
	imdb.DB[blockHash] = b
	imdb.lock.Unlock()
	return nil
}
