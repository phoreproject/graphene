package main

import (
	"fmt"
	"github.com/phoreproject/synapse/cfg"
	"github.com/phoreproject/synapse/shard/config"
	"github.com/phoreproject/synapse/shard/module"
	"github.com/phoreproject/synapse/utils"

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

	sa, err := module.NewShardApp(shardConfig)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Println("started shard")

	err = sa.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
