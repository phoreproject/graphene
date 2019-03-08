package db

import (
	"errors"
	"fmt"
	"sync"

	"github.com/phoreproject/prysm/shared/ssz"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// InMemoryDB is a very basic block database.
type InMemoryDB struct {
	DB            map[chainhash.Hash]primitives.Block
	AttestationDB map[uint32]primitives.Attestation
	headState     primitives.State
	headBlock     chainhash.Hash
	lock          *sync.Mutex
}

// NewInMemoryDB initializes a new in-memory DB
func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		DB:            make(map[chainhash.Hash]primitives.Block),
		AttestationDB: make(map[uint32]primitives.Attestation),
		lock:          new(sync.Mutex),
	}
}

// GetBlockForHash is a database lookup function
func (db *InMemoryDB) GetBlockForHash(h chainhash.Hash) (*primitives.Block, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	out, found := db.DB[h]
	if !found {
		return nil, fmt.Errorf("could not find block with hash")
	}
	return &out, nil
}

// SetBlock adds the block to storage
func (db *InMemoryDB) SetBlock(b primitives.Block) error {
	db.lock.Lock()
	blockHash, err := ssz.TreeHash(b)
	if err != nil {
		return err
	}
	db.DB[blockHash] = b
	db.lock.Unlock()
	return nil
}

// GetLatestAttestation gets the latest attestation from a validator.
func (db *InMemoryDB) GetLatestAttestation(validator uint32) (*primitives.Attestation, error) {
	if att, found := db.AttestationDB[validator]; found {
		return &att, nil
	}
	return nil, errors.New("could not find attestation for validator")
}

// SetLatestAttestationIfNeeded sets the latest attestation received from a validator.
func (db *InMemoryDB) SetLatestAttestationIfNeeded(validator uint32, att primitives.Attestation) error {
	if a, found := db.AttestationDB[validator]; found && a.Data.Slot >= att.Data.Slot {
		return nil
	}
	db.AttestationDB[validator] = att
	return nil
}

// Close closes the database.
func (db *InMemoryDB) Close() error {
	return nil
}

// SetHeadState sets the head state.
func (db *InMemoryDB) SetHeadState(state primitives.State) error {
	return nil
}

// GetHeadState gets the head state.
func (db *InMemoryDB) GetHeadState() (*primitives.State, error) {
	return nil, errors.New("no head state yet")
}

// SetHeadBlock sets the head block.
func (db *InMemoryDB) SetHeadBlock(h chainhash.Hash) error {
	return nil
}

// GetHeadBlock gets the head block.
func (db *InMemoryDB) GetHeadBlock() (*chainhash.Hash, error) {
	return nil, errors.New("no head block yet")
}

// GetBlockNode gets the block node with slot.
func (db *InMemoryDB) GetBlockNode(chainhash.Hash) (*BlockNodeDisk, error) {
	return nil, errors.New("not implemented")
}

// GetBlockState gets the state for a block.
func (db *InMemoryDB) GetBlockState(chainhash.Hash) (*primitives.State, error) {
	return nil, errors.New("not implemented")
}

// GetFinalizedHead gets the finalized head block for a chain.
func (db *InMemoryDB) GetFinalizedHead() (*chainhash.Hash, error) {
	return nil, errors.New("not implemented")
}

// GetJustifiedHead gets the justified head block for a chain.
func (db *InMemoryDB) GetJustifiedHead() (*chainhash.Hash, error) {
	return nil, errors.New("not implemented")
}

// SetBlockNode sets the block node in the database.
func (db *InMemoryDB) SetBlockNode(BlockNodeDisk) error {
	return nil
}

// SetBlockState sets the block state in the database.
func (db *InMemoryDB) SetBlockState(chainhash.Hash, primitives.State) error {
	return nil
}

// SetFinalizedHead sets the finalized head for the chain in the database.
func (db *InMemoryDB) SetFinalizedHead(chainhash.Hash) error {
	return nil
}

// SetJustifiedHead sets the justified head for the chain in the database.
func (db *InMemoryDB) SetJustifiedHead(chainhash.Hash) error {
	return nil
}

// DeleteStateForBlock deletes state for a certain block.
func (db *InMemoryDB) DeleteStateForBlock(chainhash.Hash) error {
	return nil
}

// GetGenesisTime gets the genesis time for the chain represented by this database.
func (db *InMemoryDB) GetGenesisTime() (uint64, error) {
	return 0, errors.New("in-memory database does not keep track of genesis time")
}

// SetGenesisTime sets the genesis time for the chain represented by this database.
func (db *InMemoryDB) SetGenesisTime(uint64) error {
	return nil
}

var _ Database = &InMemoryDB{}
