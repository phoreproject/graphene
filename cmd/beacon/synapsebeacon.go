package main

import (
	"fmt"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/module"
	"github.com/phoreproject/synapse/cfg"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/utils"
	"os"
	"strconv"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
)

const clientVersion = "0.2.6"

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

	logger.WithField("version", clientVersion).Info("initializing client")

	initialPeers, err := p2p.ParseInitialConnections(beaconConfig.InitialConnections)
	if err != nil {
		logger.Fatal(err)
	}

	f, err := os.Open(beaconConfig.ChainCFG)
	if err != nil {
		logger.Fatal(err)
	}

	appConfig, err := module.ReadChainFileToConfig(f)
	if err != nil {
		logger.Fatal(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	appConfig.ListeningAddress = beaconConfig.P2PListen
	appConfig.RPCAddress = beaconConfig.RPCListen
	appConfig.DiscoveryOptions.PeerAddresses = append(appConfig.DiscoveryOptions.PeerAddresses, initialPeers...)
	appConfig.DataDirectory = beaconConfig.DataDir

	appConfig.Resync = beaconConfig.Resync
	if appConfig.GenesisTime == 0 {
		if beaconConfig.GenesisTime == "" {
			appConfig.GenesisTime = uint64(utils.Now().Unix())
		} else {
			genesisTimeString := beaconConfig.GenesisTime
			if strings.HasPrefix(genesisTimeString, "+") {
				offsetString := genesisTimeString[1:]

				offset, err := strconv.Atoi(offsetString)
				if err != nil {
					panic(fmt.Errorf("invalid offset (should be number): %s", offsetString))
				}

				appConfig.GenesisTime = uint64(utils.Now().Add(time.Duration(offset) * time.Second).Unix())
			} else if strings.HasPrefix(genesisTimeString, "-") {
				offsetString := genesisTimeString[1:]

				offset, err := strconv.Atoi(offsetString)
				if err != nil {
					panic(fmt.Errorf("invalid offset (should be number): %s", offsetString))
				}

				appConfig.GenesisTime = uint64(utils.Now().Add(time.Duration(-offset) * time.Second).Unix())
			} else {
				offset, err := strconv.Atoi(genesisTimeString)
				if err != nil {
					panic(fmt.Errorf("invalid genesis time (should be number): %s", genesisTimeString))
				}

				appConfig.GenesisTime = uint64(offset)
			}
		}
	}

	changed, newLimit, err := utils.ManageFdLimit()
	if err != nil {
		panic(err)
	}
	if changed {
		logger.Infof("changed open file limit to: %d", newLimit)
	}

	a := module.NewBeaconApp(*appConfig)
	err = a.Run()
	if err != nil {
		panic(err)
	}
}
