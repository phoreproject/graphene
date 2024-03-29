package main

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/phoreproject/synapse/cfg"
	"github.com/phoreproject/synapse/relayer/config"
	"github.com/phoreproject/synapse/relayer/module"
	"github.com/phoreproject/synapse/utils"
	logger "github.com/sirupsen/logrus"
)

func main() {
	relayerConfig := config.Options{}
	globalConfig := cfg.GlobalOptions{}
	err := cfg.LoadFlags(&relayerConfig, &globalConfig)
	if err != nil {
		logger.Fatal(err)
	}

	utils.CheckNTP()

	lvl, err := logger.ParseLevel(globalConfig.LogLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logger.SetLevel(lvl)

	logger.StandardLogger().SetFormatter(&logger.TextFormatter{
		ForceColors: globalConfig.ForceColors,
	})

	changed, newLimit, err := utils.ManageFdLimit()
	if err != nil {
		logger.Fatal(err)
	}
	if changed {
		logger.Infof("changed open file limit to: %d", newLimit)
	}

	a, err := module.NewRelayerModule(relayerConfig, []peer.AddrInfo{})
	if err != nil {
		logger.Fatal(err)
	}

	err = a.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
