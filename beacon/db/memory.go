package db

import (
	"errors"
	"fmt"
	"sync"

	"github.com/phoreproject/prysm/shared/ssz"

	"github.com/phoreproject/synapse/beacon/primitives"
	"github.com/phoreproject/synapse/chainhash"
)

// InMemoryDB is a very basic block database.
type InMemoryDB struct {
	DB            map[chainhash.Hash]primitives.Block
	AttestationDB map[uint32]primitives.Attestation
	lock          *sync.Mutex
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

// GetLatestAttestation gets the latest attestation from a validator.
func (imdb *InMemoryDB) GetLatestAttestation(validator uint32) (primitives.Attestation, error) {
	if att, found := imdb.AttestationDB[validator]; found {
		return att, nil
	}
	return primitives.Attestation{}, errors.New("could not find attestation for validator")
}

// SetLatestAttestation sets the latest attestation received from a validator.
func (imdb *InMemoryDB) SetLatestAttestation(validator uint32, att primitives.Attestation) error {
	imdb.AttestationDB[validator] = att
	return nil
}
