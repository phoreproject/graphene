package main

import (
	"flag"
	"fmt"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/utils"
	"github.com/phoreproject/synapse/validator/app"

	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	logrus.Info("Starting validator manager")
	beaconHost := flag.String("beaconhost", ":11782", "the address to connect to the beacon node")
	validators := flag.String("validators", "", "validators to manage (id separated by commas) (ex. \"1,2,3\")")
	networkID := flag.String("networkid", "testnet", "networkID to use when starting network")
	rootkey := flag.String("rootkey", "testnet", "root key to run validators")
	flag.Parse()

	utils.CheckNTP()

	logrus.WithField("validators", *validators).Debug("running with validators")

	logrus.Info("connecting to blockchain RPC")

	blockchainConn, err := grpc.Dial(*beaconHost, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	networkConfig, found := config.NetworkIDs[*networkID]
	if !found {
		panic(fmt.Errorf("could not find network config %s", *networkID))
	}

	c := app.ValidatorConfig{
		BlockchainConn: blockchainConn,
		RootKey:        *rootkey,
		NetworkConfig:  &networkConfig,
	}
	c.ParseValidatorIndices(*validators)

	a := app.NewValidatorApp(c)
	err = a.Run()
	if err != nil {
		panic(err)
	}
}
