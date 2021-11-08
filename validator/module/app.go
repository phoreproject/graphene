package module

import (
	"bytes"
	"context"
	"fmt"

	beaconconfig "github.com/phoreproject/graphene/beacon/config"
	"github.com/phoreproject/graphene/utils"
	"github.com/phoreproject/graphene/validator/config"
	"google.golang.org/grpc"

	"github.com/phoreproject/graphene/primitives"

	"github.com/phoreproject/graphene/pb"
	logger "github.com/sirupsen/logrus"

	"github.com/phoreproject/graphene/validator"
)

// ValidatorApp is the app to run the validator runtime.
type ValidatorApp struct {
	config config.ValidatorConfig
	ctx    context.Context
	cancel context.CancelFunc
}

// NewValidatorApp creates a new validator app from the config.
func NewValidatorApp(options config.Options) (*ValidatorApp, error) {
	beaconAddr, err := utils.MultiaddrStringToDialString(options.BeaconRPC)
	if err != nil {
		return nil, err
	}

	shardAddr, err := utils.MultiaddrStringToDialString(options.ShardRPC)
	if err != nil {
		return nil, err
	}

	beaconConn, err := grpc.Dial(beaconAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	shardConn, err := grpc.Dial(shardAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	networkConfig, found := beaconconfig.NetworkIDs[options.NetworkID]
	if !found {
		return nil, fmt.Errorf("could not find network config %s", options.NetworkID)
	}

	c := config.ValidatorConfig{
		BeaconConn:    beaconConn,
		ShardConn:     shardConn,
		RootKey:       options.RootKey,
		NetworkConfig: &networkConfig,
	}
	c.ParseValidatorIndices(options.Validators)

	ctx, cancel := context.WithCancel(context.Background())
	return &ValidatorApp{
		config: c,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Run starts the validator app.
func (v *ValidatorApp) Run() error {
	blockchainRPC := pb.NewBlockchainRPCClient(v.config.BeaconConn)

	shardRPC := pb.NewShardRPCClient(v.config.ShardConn)

	keystore := validator.NewRootKeyStore(v.config.RootKey)

	logger.Info("Checking validator public keys...")

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

	logger.Info("Validators successfully verified!")

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
