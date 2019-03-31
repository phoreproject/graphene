package db

import (
	"bytes"
	"encoding/binary"
	"github.com/libp2p/go-libp2p-crypto"
	"log"
	"runtime"

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
	if runtime.GOOS == "windows" {
		opts.Truncate = true
		opts.ValueLogFileSize = 1024 * 1024
		// Not sure how this option takes effect.
		//opts.CompactL0OnClose = false
	}
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}

	return &BadgerDB{
		db: db,
	}
}

var blockPrefix = []byte("block")

// Flush flushes all block data.
func (b *BadgerDB) Flush() error {
	return b.db.DropAll()
}

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
	blockProto := new(pb.Block)
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
	attProto := new(pb.Attestation)
	err = proto.Unmarshal(attestationBytesCopy, attProto)
	if err != nil {
		return nil, err
	}

	return primitives.AttestationFromProto(attProto)
}

// SetLatestAttestationIfNeeded sets the latest attestation from a validator.
func (b *BadgerDB) SetLatestAttestationIfNeeded(validator uint32, attestation primitives.Attestation) error {
	var validatorBytes [4]byte
	binary.BigEndian.PutUint32(validatorBytes[:], validator)
	key := append(attestationPrefix, validatorBytes[:]...)

	attProto := attestation.ToProto()
	attSer, err := proto.Marshal(attProto)
	if err != nil {
		return err
	}

	return b.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		// if there's no attestation yet, set this as the latest
		if err == badger.ErrKeyNotFound {
			return txn.Set(key, attSer)
		}
		if err != nil {
			return err
		}

		// if there is an attestation, set if the slot is greater
		return item.Value(func(val []byte) error {
			currentAttProto := new(pb.Attestation)
			err := proto.Unmarshal(val, currentAttProto)
			if err != nil {
				return err
			}
			currentAtt, err := primitives.AttestationFromProto(currentAttProto)
			if err != nil {
				return err
			}
			// if the current attestation has a slot gt/eq to the incoming one,
			// keep the current one.
			if currentAtt.Data.Slot >= attestation.Data.Slot {
				return nil
			}
			return txn.Set(key, attSer)
		})
	})
}

var headBlockKey = []byte("head_block")
var justifiedHeadKey = []byte("justified_head")
var finalizedHeadKey = []byte("finalized_head")

// SetHeadBlock sets the head block for the chain.
func (b *BadgerDB) SetHeadBlock(h chainhash.Hash) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(headBlockKey, h[:])
	})
}

// GetHeadBlock gets the head block for the chain.
func (b *BadgerDB) GetHeadBlock() (*chainhash.Hash, error) {
	txn := b.db.NewTransaction(false)
	defer txn.Discard()
	i, err := txn.Get(headBlockKey)
	if err != nil {
		return nil, err
	}
	blockBytesCopy, err := i.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return chainhash.NewHash(blockBytesCopy)
}

// SetJustifiedHead sets the justified head block hash for the chain.
func (b *BadgerDB) SetJustifiedHead(h chainhash.Hash) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(justifiedHeadKey, h[:])
	})
}

// GetJustifiedHead gets the justified head block hash for the chain.
func (b *BadgerDB) GetJustifiedHead() (*chainhash.Hash, error) {
	txn := b.db.NewTransaction(false)
	defer txn.Discard()
	i, err := txn.Get(justifiedHeadKey)
	if err != nil {
		return nil, err
	}
	blockBytesCopy, err := i.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return chainhash.NewHash(blockBytesCopy)
}

// SetFinalizedHead sets the finalized head block hash for the chain.
func (b *BadgerDB) SetFinalizedHead(h chainhash.Hash) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(finalizedHeadKey, h[:])
	})
}

// GetFinalizedHead gets the finalized head block hash for the chain.
func (b *BadgerDB) GetFinalizedHead() (*chainhash.Hash, error) {
	txn := b.db.NewTransaction(false)
	defer txn.Discard()
	i, err := txn.Get(finalizedHeadKey)
	if err != nil {
		return nil, err
	}
	blockBytesCopy, err := i.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return chainhash.NewHash(blockBytesCopy)
}

var blockNodePrefix = []byte("block_node")

// BlockNodeDiskWithoutChildren is a block node stored on the disk.
type BlockNodeDiskWithoutChildren struct {
	Hash   chainhash.Hash
	Height uint64
	Slot   uint64
	Parent chainhash.Hash
}

// SetBlockNode sets a block node in the database.
func (b *BadgerDB) SetBlockNode(node BlockNodeDisk) error {
	nodeWithoutChildren := BlockNodeDiskWithoutChildren{
		Hash:   node.Hash,
		Height: node.Height,
		Slot:   node.Slot,
		Parent: node.Parent,
	}

	children := node.Children

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, nodeWithoutChildren)
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.BigEndian, uint32(len(children)))
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.BigEndian, children)
	if err != nil {
		return err
	}
	key := append(blockNodePrefix, node.Hash[:]...)
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, buf.Bytes())
	})
}

// GetBlockNode gets a block node from the database.
func (b *BadgerDB) GetBlockNode(h chainhash.Hash) (*BlockNodeDisk, error) {
	key := append(blockNodePrefix, h[:]...)
	txn := b.db.NewTransaction(false)
	defer txn.Discard()
	i, err := txn.Get(key)
	if err != nil {
		return nil, err
	}
	blockNodeBytesCopy, err := i.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	blockNode := new(BlockNodeDiskWithoutChildren)
	r := bytes.NewReader(blockNodeBytesCopy)

	err = binary.Read(r, binary.BigEndian, blockNode)
	if err != nil {
		return nil, err
	}

	var length uint32
	err = binary.Read(r, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	blockNodeChildren := make([]chainhash.Hash, length)

	err = binary.Read(r, binary.BigEndian, blockNodeChildren)
	if err != nil {
		return nil, err
	}

	return &BlockNodeDisk{
		Hash:     blockNode.Hash,
		Height:   blockNode.Height,
		Slot:     blockNode.Slot,
		Parent:   blockNode.Parent,
		Children: blockNodeChildren,
	}, nil
}

var blockStatePrefix = []byte("block_state")

// GetBlockState gets the block state from the database.
func (b *BadgerDB) GetBlockState(blockHash chainhash.Hash) (*primitives.State, error) {
	key := append(blockStatePrefix, blockHash[:]...)
	txn := b.db.NewTransaction(false)
	defer txn.Discard()
	i, err := txn.Get(key)
	if err != nil {
		return nil, err
	}
	blockStateBytesCopy, err := i.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	s := new(pb.State)
	err = proto.Unmarshal(blockStateBytesCopy, s)
	if err != nil {
		return nil, err
	}

	return primitives.StateFromProto(s)
}

// SetBlockState sets the block state for a specific block.
func (b *BadgerDB) SetBlockState(blockHash chainhash.Hash, state primitives.State) error {
	stateBytes, err := proto.Marshal(state.ToProto())
	if err != nil {
		return err
	}
	key := append(blockStatePrefix, blockHash[:]...)
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, stateBytes)
	})
}

// DeleteStateForBlock deletes the state for a certain block.
func (b *BadgerDB) DeleteStateForBlock(h chainhash.Hash) error {
	key := append(blockStatePrefix, h[:]...)
	return b.db.Update(func(tx *badger.Txn) error {
		err := tx.Delete(key)
		if err != nil {
			return err
		}
		return nil
	})
}

// Close closes the database.
func (b *BadgerDB) Close() error {
	return b.db.Close()
}

var genesisTimeKey = []byte("genesis_time")

// GetGenesisTime gets the genesis time of the blockchain.
func (b *BadgerDB) GetGenesisTime() (uint64, error) {
	txn := b.db.NewTransaction(false)
	defer txn.Discard()
	i, err := txn.Get(genesisTimeKey)
	if err != nil {
		return 0, err
	}
	genesisTimeBytes, err := i.ValueCopy(nil)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(genesisTimeBytes), nil
}

// SetGenesisTime sets the head block for the chain.
func (b *BadgerDB) SetGenesisTime(time uint64) error {
	genesisTimeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(genesisTimeBytes[:], time)
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(genesisTimeKey, genesisTimeBytes[:])
	})
}

var hostPrivateKeyKey = []byte("host_key")

// GetHostKey gets the key used by the P2P interface for identity.
func (b *BadgerDB) GetHostKey() (crypto.PrivKey, error) {
	txn := b.db.NewTransaction(false)
	defer txn.Discard()
	i, err := txn.Get(hostPrivateKeyKey)
	if err != nil {
		return nil, err
	}
	privKeyBytes, err := i.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return crypto.UnmarshalPrivateKey(privKeyBytes)
}

// SetHostKey sets the key used by the P2P interface for identity.
func (b *BadgerDB) SetHostKey(key crypto.PrivKey) error {
	privKeyBytes, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		return err
	}

	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(hostPrivateKeyKey, privKeyBytes)
	})
}
