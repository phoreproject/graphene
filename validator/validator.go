package validator

import (
	"context"

	"github.com/phoreproject/graphene/beacon/config"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/graphene/primitives"

	"github.com/phoreproject/graphene/pb"
)

// Validator is a single validator to keep track of
type Validator struct {
	keystore           Keystore
	blockchainRPC      pb.BlockchainRPCClient
	shardRPC           pb.ShardRPCClient
	id                 uint32
	logger             *logrus.Entry
	config             *config.Config
	forkData           *primitives.ForkData
	attestationRequest chan attestationAssignment
	proposerRequest    chan proposerAssignment
	ctx                context.Context
}

// NewValidator gets a validator
func NewValidator(ctx context.Context, keystore Keystore, blockchainRPC pb.BlockchainRPCClient, shardRPC pb.ShardRPCClient, id uint32, c *config.Config, f *primitives.ForkData) (*Validator, error) {
	v := &Validator{
		keystore:           keystore,
		blockchainRPC:      blockchainRPC,
		shardRPC:           shardRPC,
		id:                 id,
		config:             c,
		forkData:           f,
		attestationRequest: make(chan attestationAssignment),
		proposerRequest:    make(chan proposerAssignment),
		ctx:                ctx,
	}
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	v.logger = l.WithField("validator", id)
	return v, nil
}
