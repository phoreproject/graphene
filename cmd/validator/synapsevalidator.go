package main

import (
	"fmt"
	beaconconfig "github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/cfg"
	"github.com/phoreproject/synapse/validator/config"

	"github.com/phoreproject/synapse/utils"
	"github.com/phoreproject/synapse/validator/module"

	logger "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

func main() {
	validatorConfig := config.Options{}
	globalConfig := cfg.GlobalOptions{}
	err := cfg.LoadFlags(&validatorConfig, &globalConfig)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("Starting validator manager")

	utils.CheckNTP()

	logger.WithField("validators", validatorConfig.Validators).Debug("running with validators")

	logger.Info("connecting to blockchain RPC")

	beaconAddr, err := utils.MultiaddrStringToDialString(validatorConfig.BeaconRPC)
	if err != nil {
		logger.Fatal(err)
	}

	shardAddr, err := utils.MultiaddrStringToDialString(validatorConfig.ShardRPC)
	if err != nil {
		logger.Fatal(err)
	}

	beaconConn, err := grpc.Dial(beaconAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatal(err)
	}

	shardConn, err := grpc.Dial(shardAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatal(err)
	}

	networkConfig, found := beaconconfig.NetworkIDs[validatorConfig.NetworkID]
	if !found {
		logger.Fatal(fmt.Errorf("could not find network config %s", validatorConfig.NetworkID))
	}

	c := config.ValidatorConfig{
		BeaconConn:    beaconConn,
		ShardConn:     shardConn,
		RootKey:       validatorConfig.RootKey,
		NetworkConfig: &networkConfig,
	}
	c.ParseValidatorIndices(validatorConfig.Validators)

	a := module.NewValidatorApp(c)
	err = a.Run()
	if err != nil {
		panic(err)
	}
}
