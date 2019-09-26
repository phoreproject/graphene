package module

import (
	"bytes"
	"context"
	"fmt"
	"github.com/phoreproject/synapse/validator/config"

	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/pb"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/validator"
)

// ValidatorApp is the app to run the validator runtime.
type ValidatorApp struct {
	config config.ValidatorConfig
	ctx    context.Context
	cancel context.CancelFunc
}

var log = logrus.New()

// NewValidatorApp creates a new validator app from the config.
func NewValidatorApp(config config.ValidatorConfig) *ValidatorApp {
	ctx, cancel := context.WithCancel(context.Background())
	return &ValidatorApp{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Run starts the validator app.
func (v *ValidatorApp) Run() error {
	blockchainRPC := pb.NewBlockchainRPCClient(v.config.BeaconConn)

	shardRPC := pb.NewShardRPCClient(v.config.ShardConn)

	keystore := validator.NewRootKeyStore(v.config.RootKey)

	log.Info("Checking validator public keys...")

	for _, val := range v.config.ValidatorIndices {
		validatorProto, err := blockchainRPC.GetValidatorInformation(v.ctx, &pb.GetValidatorRequest{ID: uint32(val)})
		if err != nil {
			return err
		}

		v, err := primitives.ValidatorFromProto(validatorProto)
		if err != nil {
			return err
		}

		expectedPublicKey := keystore.GetPublicKeyForValidator(val).Serialize()

		if !bytes.Equal(expectedPublicKey[:], v.Pubkey[:]) {
			return fmt.Errorf("validator %d public key did not match current validator set", val)
		}
	}

	log.Info("Validators successfully verified!")

	vm, err := validator.NewManager(v.ctx, blockchainRPC, shardRPC, v.config.ValidatorIndices, keystore, v.config.NetworkConfig)
	if err != nil {
		return err
	}

	return vm.Start()
}

// Exit exits the validator app.
func (v *ValidatorApp) Exit() {
	v.cancel()
}
