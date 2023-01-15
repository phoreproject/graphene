package main

import (
	"github.com/phoreproject/synapse/cfg"
	"github.com/phoreproject/synapse/explorer/config"

	"github.com/phoreproject/synapse/explorer/module"
	"github.com/phoreproject/synapse/utils"

	logger "github.com/sirupsen/logrus"
)

func main() {
	explorerConfig := config.Options{}
	globalConfig := cfg.GlobalOptions{}
	err := cfg.LoadFlags(&explorerConfig, &globalConfig)
	if err != nil {
		logger.Fatal(err)
	}

	logger.StandardLogger().SetFormatter(&logger.TextFormatter{
		ForceColors: globalConfig.ForceColors,
	})

	utils.CheckNTP()

	a, err := module.NewExplorerApp(explorerConfig)
	if err != nil {
		logger.Fatal(err)
	}

	err = a.Run()
	if err != nil {
		logger.Fatal(err)
	}

	a.Exit()
}
