package main

import (
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/module"
	"github.com/phoreproject/synapse/cfg"
	"github.com/phoreproject/synapse/utils"
	logger "github.com/sirupsen/logrus"
	/*
		Uncomment to enable memory profiling. To use it, run synapsebeacon, then
		  curl -sK -v http://localhost:9000/debug/pprof/heap > heap
		  go tool pprof heap
		or to see all allocated memory
		  go tool pprof --alloc_space heap
		in go pprof, use 'top' command to see the top memory usage
	*///"net/http"
	//_ "net/http/pprof"
)

func main() {
	//go http.ListenAndServe("localhost:9000", nil)

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
