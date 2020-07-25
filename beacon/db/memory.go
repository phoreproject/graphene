package db

import (
	"errors"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/prysmaticlabs/go-ssz"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// InMemoryDB is a very basic block database.
type InMemoryDB struct {
	DB             map[chainhash.Hash]primitives.Block
	AttestationDB  map[uint32]primitives.Attestation
	head           chainhash.Hash
	justified      chainhash.Hash
	blockNodes     map[chainhash.Hash]BlockNodeDisk
	statemap       map[chainhash.Hash]primitives.State
	genesisTime    uint64
	hostkey        []byte
	finalized      chainhash.Hash
	finalizedState primitives.State
	justifiedState primitives.State
	lock           *sync.Mutex
}

// NewInMemoryDB initializes a new in-memory DB
func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		DB:            make(map[chainhash.Hash]primitives.Block),
		AttestationDB: make(map[uint32]primitives.Attestation),
		blockNodes:    make(map[chainhash.Hash]BlockNodeDisk),
		statemap:      make(map[chainhash.Hash]primitives.State),
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
	blockHash, _ := ssz.HashTreeRoot(b)
	db.DB[blockHash] = b
	db.lock.Unlock()
	return nil
}

// GetLatestAttestation gets the latest attestation from a validator.
func (db *InMemoryDB) GetLatestAttestation(validator uint32, transaction ...interface{}) (*primitives.Attestation, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	if att, found := db.AttestationDB[validator]; found {
		return &att, nil
	}
	return nil, errors.New("could not find attestation for validator")
}

// SetLatestAttestationsIfNeeded sets the latest attestation received from a validator.
func (db *InMemoryDB) SetLatestAttestationsIfNeeded(validators []uint32, att primitives.Attestation, transaction ...interface{}) error {
	db.lock.Lock()
	defer db.lock.Unlock()
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
	db.lock.Lock()
	defer db.lock.Unlock()
	db.head = h
	return nil
}

// GetHeadBlock gets the head block.
func (db *InMemoryDB) GetHeadBlock(transaction ...interface{}) (*chainhash.Hash, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	return &db.head, nil
}

// GetBlockNode gets the block node with slot.
func (db *InMemoryDB) GetBlockNode(h chainhash.Hash, transaction ...interface{}) (*BlockNodeDisk, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	if b, found := db.blockNodes[h]; found {
		return &b, nil
	}
	return nil, fmt.Errorf("missing block node %s", h)
}

// GetFinalizedHead gets the finalized head block for a chain.
func (db *InMemoryDB) GetFinalizedHead(transaction ...interface{}) (*chainhash.Hash, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	return &db.finalized, nil
}

// GetJustifiedHead gets the justified head block for a chain.
func (db *InMemoryDB) GetJustifiedHead(transaction ...interface{}) (*chainhash.Hash, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	return &db.justified, nil
}

// SetBlockNode sets the block node in the database.
func (db *InMemoryDB) SetBlockNode(node BlockNodeDisk, transaction ...interface{}) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.blockNodes[node.Hash] = node
	return nil
}

// SetFinalizedHead sets the finalized head for the chain in the database.
func (db *InMemoryDB) SetFinalizedHead(h chainhash.Hash, transaction ...interface{}) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.finalized = h
	return nil
}

// SetJustifiedHead sets the justified head for the chain in the database.
func (db *InMemoryDB) SetJustifiedHead(h chainhash.Hash, transaction ...interface{}) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.justified = h
	return nil
}

// GetGenesisTime gets the genesis time for the chain represented by this database.
func (db *InMemoryDB) GetGenesisTime(transaction ...interface{}) (uint64, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	return db.genesisTime, nil
}

// SetGenesisTime sets the genesis time for the chain represented by this database.
func (db *InMemoryDB) SetGenesisTime(t uint64, transaction ...interface{}) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.genesisTime = t
	return nil
}

// GetHostKey gets the host key
func (db *InMemoryDB) GetHostKey(transaction ...interface{}) (crypto.PrivKey, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	return crypto.UnmarshalPrivateKey(db.hostkey)
}

// SetHostKey sets the host key
func (db *InMemoryDB) SetHostKey(key crypto.PrivKey, transaction ...interface{}) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	hostkey, _ := crypto.MarshalPrivateKey(key)
	db.hostkey = hostkey
	return nil
}

// GetFinalizedState gets the finalized state from the database.
func (db *InMemoryDB) GetFinalizedState(transaction ...interface{}) (*primitives.State, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	return &db.finalizedState, nil
}

// GetJustifiedState gets the justified state from the database.
func (db *InMemoryDB) GetJustifiedState(transaction ...interface{}) (*primitives.State, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	return &db.justifiedState, nil
}

// SetFinalizedState sets the finalized state
func (db *InMemoryDB) SetFinalizedState(state primitives.State, transaction ...interface{}) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.finalizedState = state
	return nil
}

// SetJustifiedState sets the justified state
func (db *InMemoryDB) SetJustifiedState(state primitives.State, transaction ...interface{}) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.justifiedState = state
	return nil
}

// TransactionalUpdate executes cb in an update transaction
func (db *InMemoryDB) TransactionalUpdate(cb func(transaction interface{}) error) error {
	return cb(nil)
}

var _ Database = &InMemoryDB{}
