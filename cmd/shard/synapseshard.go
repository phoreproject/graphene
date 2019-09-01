package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"

	"github.com/phoreproject/synapse/utils"

	"github.com/phoreproject/synapse/shard/app"

	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
)

const clientVersion = "0.2.6"

func main() {
	rpcConnect := flag.String("rpclisten", "127.0.0.1:11783", "host and port for RPC server to listen on")
	//chainconfig := flag.String("chainconfig", "testnet.json", "chain config file")
	//datadir := flag.String("datadir", "", "location to store blockchain data")
	beaconHost := flag.String("beaconhost", "127.0.0.1:11782", "host and port of beacon RPC server")

	// Logging
	level := flag.String("level", "info", "log level")
	flag.Parse()

	utils.CheckNTP()

	lvl, err := logrus.ParseLevel(*level)
	if err != nil {
		panic(err)
	}
	logrus.SetLevel(lvl)

	logger.WithField("version", clientVersion).Info("initializing shard manager")

	changed, newLimit, err := utils.ManageFdLimit()
	if err != nil {
		panic(err)
	}
	if changed {
		logger.Infof("changed open file limit to: %d", newLimit)
	}

	cc, err := grpc.Dial(*beaconHost, grpc.WithInsecure())
	if err != nil {
		err = fmt.Errorf("could not connect to beacon host at %s with error: %s", *beaconHost, err)
		panic(err)
	}

	sc := app.ShardConfig{
		BeaconConn:  cc,
		RPCProtocol: "tcp",
		RPCAddress:  *rpcConnect,
	}

	sa := app.NewShardApp(sc)

	err = sa.Run()
	if err != nil {
		panic(err)
	}
}
