package db

import (
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// Database is a very basic interface for pluggable
// databases.
type Database interface {
	GetBlockForHash(h chainhash.Hash) (*primitives.Block, error)
	SetBlock(b primitives.Block) error
	GetLatestAttestation(validator uint32) (*primitives.Attestation, error)
	SetLatestAttestation(validator uint32, attestation primitives.Attestation) error
	SetHeadState(state primitives.State) error
	GetHeadState() (*primitives.State, error)
	SetHeadBlock(h chainhash.Hash) error
	GetHeadBlock() (*chainhash.Hash, error)
	Close()
}

// to load chain, we need:
// - all block nodes
// - some block nodes states
// - chain tip
