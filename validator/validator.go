package validator

import (
	"time"

	"github.com/inconshreveable/log15"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
)

// Validator is a single validator to keep track of
type Validator struct {
	secretKey *bls.SecretKey
	rpc       *rpc.BlockchainRPCClient
	id        uint32
}

// NewValidator gets a validator
func NewValidator(key *bls.SecretKey, rpc *rpc.BlockchainRPCClient, id uint32) *Validator {
	return &Validator{secretKey: key, rpc: rpc, id: id}
}

// RunValidator keeps track of assignments and creates/signs attestations as needed.
func (v *Validator) RunValidator() error {
	log15.Info("Running validator", "validator", v.id)

	t := time.NewTicker(time.Second)

	for {
		<-t.C

	}
}
