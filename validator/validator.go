package validator

import (
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/pb"
)

// Validator is a single validator to keep track of
type Validator struct {
	keystore           Keystore
	blockchainRPC      pb.BlockchainRPCClient
	p2pRPC             pb.P2PRPCClient
	id                 uint32
	logger             *logrus.Entry
	mempool            *mempool
	config             *config.Config
	forkData           *primitives.ForkData
	attestationRequest chan attestationAssignment
	proposerRequest    chan proposerAssignment
}

// NewValidator gets a validator
func NewValidator(keystore Keystore, blockchainRPC pb.BlockchainRPCClient, p2pRPC pb.P2PRPCClient, id uint32, mempool *mempool, c *config.Config, f *primitives.ForkData) *Validator {
	v := &Validator{
		keystore:           keystore,
		blockchainRPC:      blockchainRPC,
		p2pRPC:             p2pRPC,
		id:                 id,
		mempool:            mempool,
		config:             c,
		forkData:           f,
		attestationRequest: make(chan attestationAssignment),
		proposerRequest:    make(chan proposerAssignment),
	}
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	v.logger = l.WithField("validator", id)
	return v
}

// RunValidator keeps track of assignments and creates/signs attestations as needed.
func (v *Validator) RunValidator() error {
	for {
		p := <-v.proposerRequest
		err := v.proposeBlock(p)
		if err != nil {
			return err
		}
	}
}
