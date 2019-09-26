package main

import (
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/phoreproject/synapse/cfg"
	"github.com/phoreproject/synapse/shard/config"
	"github.com/phoreproject/synapse/utils"
	"google.golang.org/grpc"

	"github.com/phoreproject/synapse/shard/module"

	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
)

const clientVersion = "0.2.6"

func main() {
	shardConfig := config.Options{}
	globalConfig := cfg.GlobalOptions{}
	err := cfg.LoadFlags(&shardConfig, &globalConfig)
	if err != nil {
		logger.Fatal(err)
	}

	utils.CheckNTP()

	lvl, err := logrus.ParseLevel(globalConfig.LogLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logrus.SetLevel(lvl)

	logger.WithField("version", clientVersion).Info("initializing shard manager")

	changed, newLimit, err := utils.ManageFdLimit()
	if err != nil {
		logger.Fatal(err)
	}
	if changed {
		logger.Infof("changed open file limit to: %d", newLimit)
	}

	beaconAddr, err := utils.MultiaddrStringToDialString(shardConfig.BeaconRPC)
	if err != nil {
		logger.Fatal(err)
	}

	cc, err := grpc.Dial(beaconAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("could not connect to beacon host at %s with error: %s", shardConfig.BeaconRPC, err)
	}

	ma, err := multiaddr.NewMultiaddr(shardConfig.RPCListen)
	if err != nil {
		logger.Fatal(err)
	}

	addr, err := manet.ToNetAddr(ma)
	if err != nil {
		logger.Fatal(err)
	}

	sc := config.ShardConfig{
		BeaconConn:  cc,
		RPCProtocol: addr.Network(),
		RPCAddress:  addr.String(),
	}

	sa := module.NewShardApp(sc)

	err = sa.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
