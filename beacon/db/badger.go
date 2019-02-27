package db

import (
	"encoding/binary"
	"log"

	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/pb"

	"github.com/golang/protobuf/proto"

	"github.com/dgraph-io/badger"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

var _ Database = (*BadgerDB)(nil)

// BadgerDB is a wrapper around the badger database to provide functions for
// storing blocks and attestations.
type BadgerDB struct {
	db *badger.DB
}

// NewBadgerDB initializes the badger database with the supplied directories.
func NewBadgerDB(databaseDir string, databaseValueDir string) *BadgerDB {
	opts := badger.DefaultOptions
	opts.Dir = databaseDir
	opts.ValueDir = databaseValueDir
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}

	return &BadgerDB{
		db: db,
	}
}

var blockPrefix = []byte("block")

// GetBlockForHash gets a block for a certain block hash.
func (b *BadgerDB) GetBlockForHash(h chainhash.Hash) (*primitives.Block, error) {
	key := append(blockPrefix, h[:]...)
	txn := b.db.NewTransaction(false)
	defer txn.Discard()
	i, err := txn.Get(key)
	if err != nil {
		return nil, err
	}
	blockBytesCopy, err := i.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	var blockProto *pb.Block
	err = proto.Unmarshal(blockBytesCopy, blockProto)
	if err != nil {
		return nil, err
	}

	return primitives.BlockFromProto(blockProto)
}

// SetBlock sets a block for a certain block hash.
func (b *BadgerDB) SetBlock(block primitives.Block) error {
	blockHash, err := ssz.TreeHash(block)
	if err != nil {
		return err
	}

	key := append(blockPrefix, blockHash[:]...)

	blockProto := block.ToProto()
	blockSer, err := proto.Marshal(blockProto)
	if err != nil {
		return err
	}

	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, blockSer)
	})
}

var attestationPrefix = []byte("att")

// GetLatestAttestation gets the latest attestation from a validator.
func (b *BadgerDB) GetLatestAttestation(validator uint32) (*primitives.Attestation, error) {
	var validatorBytes [4]byte
	binary.BigEndian.PutUint32(validatorBytes[:], validator)
	key := append(attestationPrefix, validatorBytes[:]...)
	txn := b.db.NewTransaction(false)
	defer txn.Discard()
	i, err := txn.Get(key)
	if err != nil {
		return nil, err
	}
	attestationBytesCopy, err := i.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	var attProto *pb.Attestation
	err = proto.Unmarshal(attestationBytesCopy, attProto)
	if err != nil {
		return nil, err
	}

	return primitives.AttestationFromProto(attProto)
}

// SetLatestAttestation sets the latest attestation from a validator.
func (b *BadgerDB) SetLatestAttestation(validator uint32, attestation primitives.Attestation) error {
	var validatorBytes [4]byte
	binary.BigEndian.PutUint32(validatorBytes[:], validator)
	key := append(attestationPrefix, validatorBytes[:]...)

	attProto := attestation.ToProto()
	attSer, err := proto.Marshal(attProto)
	if err != nil {
		return err
	}

	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, attSer)
	})
}

// Close closes the database.
func (b *BadgerDB) Close() {
	b.db.Close()
}
