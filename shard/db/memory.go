package db

import (
	"fmt"
	"sync"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/go-ssz"
)

// MemoryBlockDB is a block database stored in memory.
type MemoryBlockDB struct {
	blocks     map[chainhash.Hash]primitives.ShardBlock
	blockNodes map[chainhash.Hash]ShardBlockNodeDisk

	finalizedHead chainhash.Hash
	tip           chainhash.Hash

	shardCodes map[uint64][]byte

	lock sync.Mutex
}

// SetBlockNode sets a block node in the database.
func (m *MemoryBlockDB) SetBlockNode(n *ShardBlockNodeDisk) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.blockNodes[n.BlockHash] = *n
	return nil
}

// SetFinalizedHead sets the finalized head in the database.
func (m *MemoryBlockDB) SetFinalizedHead(c chainhash.Hash) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.finalizedHead = c
	return nil
}

// GetFinalizedHead returns the hash of the finalized head.
func (m *MemoryBlockDB) GetFinalizedHead() (*chainhash.Hash, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	return &m.finalizedHead, nil
}

// SetChainTip sets the tip of the chain.
func (m *MemoryBlockDB) SetChainTip(c chainhash.Hash) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.tip = c
	return nil
}

// GetChainTip gets the tip of the chain.
func (m *MemoryBlockDB) GetChainTip() (*chainhash.Hash, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	return &m.tip, nil
}

// NewMemoryBlockDB creates a new block database stored in memory.
func NewMemoryBlockDB() *MemoryBlockDB {
	return &MemoryBlockDB{
		blocks: make(map[chainhash.Hash]primitives.ShardBlock),
	}
}

// GetBlockForHash gets a block from the database.
func (m *MemoryBlockDB) GetBlockForHash(h chainhash.Hash) (*primitives.ShardBlock, error) {
	if b, found := m.blocks[h]; found {
		return &b, nil
	}
	return nil, fmt.Errorf("couldn't find block in database with hash %s", h)
}

// SetBlock sets a block in the block database.
func (m *MemoryBlockDB) SetBlock(b *primitives.ShardBlock) error {
	blockHash, err := ssz.HashTreeRoot(b)
	if err != nil {
		return err
	}

	m.blocks[blockHash] = *b
	return nil
}

// GetBlockNode gets a block node from the database.
func (m *MemoryBlockDB) GetBlockNode(h chainhash.Hash) (*ShardBlockNodeDisk, error) {
	if b, found := m.blockNodes[h]; found {
		return &b, nil
	}
	return nil, fmt.Errorf("missing block node %s", h)
}

// SetCodeForShardId sets a shard code in the database.
func (m *MemoryBlockDB) SetCodeForShardId(shardID uint64, shardCode []byte) error {
	m.shardCodes[shardID] = shardCode
	return nil
}

// GetCodeForShardId gets a shard code from the database.
func (m *MemoryBlockDB) GetCodeForShardId(shardID uint64) ([]byte, error) {
	if code, found := m.shardCodes[shardID]; found {
		return code, nil
	}
	return nil, nil
}

// Close closes the database, which does nothing for an in-memory database.
func (m *MemoryBlockDB) Close() error {
	return nil
}

var _ ShardBlockDatabase = &MemoryBlockDB{}
