package db

import (
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// BlockNodeDisk is a block node stored on the disk.
type BlockNodeDisk struct {
	Hash     chainhash.Hash
	Height   uint64
	Slot     uint64
	Parent   chainhash.Hash
	Children []chainhash.Hash
}

// Database is a very basic interface for pluggable
// databases.
type Database interface {
	GetBlockForHash(h chainhash.Hash) (*primitives.Block, error)
	SetBlock(b primitives.Block) error
	GetLatestAttestation(validator uint32) (*primitives.Attestation, error)
	SetLatestAttestationsIfNeeded(validators []uint32, attestation primitives.Attestation) error
	SetHeadBlock(block chainhash.Hash) error
	GetHeadBlock() (*chainhash.Hash, error)
	SetBlockState(blockHash chainhash.Hash, state primitives.State) error
	GetBlockState(blockHash chainhash.Hash) (*primitives.State, error)
	SetBlockNode(node BlockNodeDisk) error
	GetBlockNode(h chainhash.Hash) (*BlockNodeDisk, error)
	SetJustifiedHead(h chainhash.Hash) error
	SetFinalizedHead(h chainhash.Hash) error
	GetJustifiedHead() (*chainhash.Hash, error)
	GetFinalizedHead() (*chainhash.Hash, error)
	GetGenesisTime() (uint64, error)
	SetGenesisTime(uint64) error
	GetHostKey() (crypto.PrivKey, error)
	SetHostKey(key crypto.PrivKey) error
	DeleteStateForBlock(blockHash chainhash.Hash) error
	Close() error
}

// to load chain, we need:
// - all block nodes
// - some block nodes states
// - chain tip

// populate index with all block nodes
// populate blockchainView with correct finalized/justified head
// populate statemanager with correct states for all block nodes that have state
// set head state/block
