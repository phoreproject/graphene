package db

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/phoreproject/graphene/chainhash"
	"github.com/phoreproject/graphene/primitives"
)

// BlockNodeDisk is a block node stored on the disk.
type BlockNodeDisk struct {
	Hash      chainhash.Hash
	Height    uint64
	Slot      uint64
	Parent    chainhash.Hash
	StateRoot chainhash.Hash
	Children  []chainhash.Hash
}

// Database is a very basic interface for pluggable
// databases.
type Database interface {
	GetBlockForHash(h chainhash.Hash, transaction ...interface{}) (*primitives.Block, error)
	SetBlock(b primitives.Block, transaction ...interface{}) error
	GetLatestAttestation(validator uint32, transaction ...interface{}) (*primitives.Attestation, error)
	SetLatestAttestationsIfNeeded(validators []uint32, attestation primitives.Attestation, transaction ...interface{}) error
	SetHeadBlock(block chainhash.Hash, transaction ...interface{}) error
	GetHeadBlock(transaction ...interface{}) (*chainhash.Hash, error)
	SetFinalizedState(state primitives.State, transaction ...interface{}) error
	GetFinalizedState(transaction ...interface{}) (*primitives.State, error)
	SetJustifiedState(state primitives.State, transaction ...interface{}) error
	GetJustifiedState(transaction ...interface{}) (*primitives.State, error)
	SetBlockNode(node BlockNodeDisk, transaction ...interface{}) error
	GetBlockNode(h chainhash.Hash, transaction ...interface{}) (*BlockNodeDisk, error)
	SetJustifiedHead(h chainhash.Hash, transaction ...interface{}) error
	SetFinalizedHead(h chainhash.Hash, transaction ...interface{}) error
	GetJustifiedHead(transaction ...interface{}) (*chainhash.Hash, error)
	GetFinalizedHead(transaction ...interface{}) (*chainhash.Hash, error)
	GetGenesisTime(transaction ...interface{}) (uint64, error)
	SetGenesisTime(t uint64, transaction ...interface{}) error
	GetHostKey(transaction ...interface{}) (crypto.PrivKey, error)
	SetHostKey(key crypto.PrivKey, transaction ...interface{}) error
	Close() error
	TransactionalUpdate(cb func(transaction interface{}) error) error
}

// to load chain, we need:
// - all block nodes
// - some block nodes states
// - chain tip

// populate index with all block nodes
// populate blockchainView with correct finalized/justified head
// populate statemanager with correct states for all block nodes that have state
// set head state/block
