package db

import (
	"errors"
	"fmt"
	"sync"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// InMemoryDB is a very basic block database.
type InMemoryDB struct {
	DB            map[chainhash.Hash]primitives.Block
	AttestationDB map[uint32]primitives.Attestation
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
func (db *InMemoryDB) GetBlockForHash(h chainhash.Hash, transaction ...interface{}) (*primitives.Block, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	out, found := db.DB[h]
	if !found {
		return nil, fmt.Errorf("could not find block with hash")
	}
	return &out, nil
}

// SetBlock adds the block to storage
func (db *InMemoryDB) SetBlock(b primitives.Block, transaction ...interface{}) error {
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
func (db *InMemoryDB) GetLatestAttestation(validator uint32, transaction ...interface{}) (*primitives.Attestation, error) {
	if att, found := db.AttestationDB[validator]; found {
		return &att, nil
	}
	return nil, errors.New("could not find attestation for validator")
}

// SetLatestAttestationsIfNeeded sets the latest attestation received from a validator.
func (db *InMemoryDB) SetLatestAttestationsIfNeeded(validators []uint32, att primitives.Attestation, transaction ...interface{}) error {
	for _, validator := range validators {
		if a, found := db.AttestationDB[validator]; found && a.Data.Slot >= att.Data.Slot {
			return nil
		}
		db.AttestationDB[validator] = att
	}
	return nil
}

// Close closes the database.
func (db *InMemoryDB) Close() error {
	return nil
}

// SetHeadBlock sets the head block.
func (db *InMemoryDB) SetHeadBlock(h chainhash.Hash, transaction ...interface{}) error {
	return nil
}

// GetHeadBlock gets the head block.
func (db *InMemoryDB) GetHeadBlock(transaction ...interface{}) (*chainhash.Hash, error) {
	return nil, errors.New("no head block yet")
}

// GetBlockNode gets the block node with slot.
func (db *InMemoryDB) GetBlockNode(h chainhash.Hash, transaction ...interface{}) (*BlockNodeDisk, error) {
	return nil, errors.New("not implemented")
}

// GetFinalizedHead gets the finalized head block for a chain.
func (db *InMemoryDB) GetFinalizedHead(transaction ...interface{}) (*chainhash.Hash, error) {
	return nil, errors.New("not implemented")
}

// GetJustifiedHead gets the justified head block for a chain.
func (db *InMemoryDB) GetJustifiedHead(transaction ...interface{}) (*chainhash.Hash, error) {
	return nil, errors.New("not implemented")
}

// SetBlockNode sets the block node in the database.
func (db *InMemoryDB) SetBlockNode(node BlockNodeDisk, transaction ...interface{}) error {
	return nil
}

// SetBlockState sets the block state in the database.
func (db *InMemoryDB) SetBlockState(chainhash.Hash, primitives.State) error {
	return nil
}

// SetFinalizedHead sets the finalized head for the chain in the database.
func (db *InMemoryDB) SetFinalizedHead(h chainhash.Hash, transaction ...interface{}) error {
	return nil
}

// SetJustifiedHead sets the justified head for the chain in the database.
func (db *InMemoryDB) SetJustifiedHead(h chainhash.Hash, transaction ...interface{}) error {
	return nil
}

// GetGenesisTime gets the genesis time for the chain represented by this database.
func (db *InMemoryDB) GetGenesisTime(transaction ...interface{}) (uint64, error) {
	return 0, errors.New("in-memory database does not keep track of genesis time")
}

// SetGenesisTime sets the genesis time for the chain represented by this database.
func (db *InMemoryDB) SetGenesisTime(t uint64, transaction ...interface{}) error {
	return nil
}

// GetHostKey gets the host key
func (db *InMemoryDB) GetHostKey(transaction ...interface{}) (crypto.PrivKey, error) {
	return nil, errors.New("in-memory database does not keep track of host key")
}

// SetHostKey sets the host key
func (db *InMemoryDB) SetHostKey(key crypto.PrivKey, transaction ...interface{}) error {
	return nil
}

// GetFinalizedState gets the finalized state from the database.
func (db *InMemoryDB) GetFinalizedState(transaction ...interface{}) (*primitives.State, error) {
	return nil, errors.New("in-memory database does not keep track of finalized state")
}

// GetJustifiedState gets the justified state from the database.
func (db *InMemoryDB) GetJustifiedState(transaction ...interface{}) (*primitives.State, error) {
	return nil, errors.New("in-memory database does not keep track of finalized state")
}

// SetFinalizedState sets the finalized state
func (db *InMemoryDB) SetFinalizedState(state primitives.State, transaction ...interface{}) error {
	return nil
}

// SetJustifiedState sets the justified state
func (db *InMemoryDB) SetJustifiedState(state primitives.State, transaction ...interface{}) error {
	return nil
}

// TransactionalUpdate executes cb in an update transaction
func (db *InMemoryDB) TransactionalUpdate(cb func(transaction interface{}) error) error {
	return cb(nil)
}

var _ Database = &InMemoryDB{}
