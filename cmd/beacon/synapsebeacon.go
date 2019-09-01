package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/utils"

	"github.com/phoreproject/synapse/beacon/app"

	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
)

const clientVersion = "0.2.6"

func main() {
	rpcConnect := flag.String("rpclisten", "127.0.0.1:11782", "host and port for RPC server to listen on")
	chainconfig := flag.String("chainconfig", "testnet.json", "chain config file")
	resync := flag.Bool("resync", false, "resyncs the blockchain if this is set")
	datadir := flag.String("datadir", "", "location to store blockchain data")
	genesisTime := flag.String("genesistime", "", "time to use if not specified by config")

	// P2P
	initialConnections := flag.String("connect", "", "comma separated multiaddrs")
	listen := flag.String("listen", "/ip4/0.0.0.0/tcp/11781", "specifies the address to listen on")

	// Logging
	level := flag.String("level", "info", "log level")
	flag.Parse()

	utils.CheckNTP()

	lvl, err := logrus.ParseLevel(*level)
	if err != nil {
		panic(err)
	}
	logrus.SetLevel(lvl)

	logger.WithField("version", clientVersion).Info("initializing client")

	initialPeers, err := p2p.ParseInitialConnections(*initialConnections)
	if err != nil {
		panic(err)
	}

	f, err := os.Open(*chainconfig)
	if err != nil {
		panic(err)
	}

	appConfig, err := app.ReadChainFileToConfig(f)
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	appConfig.ListeningAddress = *listen
	appConfig.RPCAddress = *rpcConnect
	appConfig.DiscoveryOptions.PeerAddresses = append(appConfig.DiscoveryOptions.PeerAddresses, initialPeers...)
	appConfig.DataDirectory = *datadir

	appConfig.Resync = *resync
	if appConfig.GenesisTime == 0 {
		if *genesisTime == "" {
			appConfig.GenesisTime = uint64(utils.Now().Unix())
		} else {
			genesisTimeString := *genesisTime
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

	a := app.NewBeaconApp(*appConfig)
	err = a.Run()
	if err != nil {
		panic(err)
	}
}
