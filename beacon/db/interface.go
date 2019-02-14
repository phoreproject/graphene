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
	GetLatestAttestation(validator uint32) (primitives.Attestation, error)
	SetLatestAttestation(validator uint32, attestation primitives.Attestation) error
}
