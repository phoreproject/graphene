package main

import (
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/module"
	"github.com/phoreproject/synapse/cfg"
	"github.com/phoreproject/synapse/utils"
	logger "github.com/sirupsen/logrus"
)

func main() {
	beaconConfig := config.Options{}
	globalConfig := cfg.GlobalOptions{}
	err := cfg.LoadFlags(&beaconConfig, &globalConfig)
	if err != nil {
		logger.Fatal(err)
	}

	utils.CheckNTP()

	lvl, err := logger.ParseLevel(globalConfig.LogLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logger.SetLevel(lvl)

	changed, newLimit, err := utils.ManageFdLimit()
	if err != nil {
		logger.Fatal(err)
	}
	if changed {
		logger.Infof("changed open file limit to: %d", newLimit)
	}

	a, err := module.NewBeaconApp(beaconConfig)
	if err != nil {
		logger.Fatal(err)
	}

	err = a.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
