package main

import (
	"github.com/phoreproject/synapse/cfg"
	"github.com/phoreproject/synapse/wallet/config"

	"github.com/phoreproject/synapse/utils"
	"github.com/phoreproject/synapse/wallet/module"

	logger "github.com/sirupsen/logrus"
)

func main() {
	walletConfig := config.Options{}
	globalConfig := cfg.GlobalOptions{}
	err := cfg.LoadFlags(&walletConfig, &globalConfig)
	if err != nil {
		logger.Fatal(err)
	}

	logger.StandardLogger().SetFormatter(&logger.TextFormatter{
		ForceColors: globalConfig.ForceColors,
	})

	utils.CheckNTP()

	a, err := module.NewWalletApp(walletConfig)
	if err != nil {
		logger.Fatal(err)
	}

	err = a.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
