package main

import (
	"flag"

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
	rootkey := flag.String("rootkey", "testnet", "root key to run validators")
	flag.Parse()

	utils.CheckNTP()

	logrus.WithField("validators", *validators).Debug("running with validators")

	logrus.Info("connecting to blockchain RPC")

	blockchainConn, err := grpc.Dial(*beaconHost, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	c := app.ValidatorConfig{
		BlockchainConn: blockchainConn,
		RootKey:        *rootkey,
	}
	c.ParseValidatorIndices(*validators)

	a := app.NewValidatorApp(c)
	a.Run()
}
