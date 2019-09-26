package main

import (
	"flag"
	"os"
	"strings"

	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/utils"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/phoreproject/synapse/explorer"

	"github.com/sirupsen/logrus"
)

func main() {
	chainconfig := flag.String("chainconfig", "testnet.json", "file of chain config")
	resync := flag.Bool("resync", false, "resyncs the blockchain if this is set")
	datadir := flag.String("datadir", "", "location to store blockchain data")
	initialConnections := flag.String("connect", "", "comma separated multiaddrs")
	listen := flag.String("listen", "/ip4/0.0.0.0/tcp/11781", "specifies the address to listen on")

	// Logging
	level := flag.String("level", "info", "log level")
	flag.Parse()

	utils.CheckNTP()

	changed, newLimit, err := utils.ManageFdLimit()
	if err != nil {
		panic(err)
	}
	if changed {
		logrus.Infof("changed ulimit to: %d", newLimit)
	}

	lvl, err := logrus.ParseLevel(*level)
	if err != nil {
		panic(err)
	}
	logrus.SetLevel(lvl)

	db, err := gorm.Open("sqlite3", "/tmp/gorm.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// we should load the keys from the validator keystore
	f, err := os.Open(*chainconfig)
	if err != nil {
		panic(err)
	}

	explorerConfig, err := explorer.ReadChainFileToConfig(f)
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	initialPeers, err := p2p.ParseInitialConnections(strings.Split(*initialConnections, ","))
	if err != nil {
		panic(err)
	}

	explorerConfig.DiscoveryOptions.PeerAddresses = append(explorerConfig.DiscoveryOptions.PeerAddresses, initialPeers...)

	explorerConfig.DataDirectory = *datadir
	explorerConfig.Resync = *resync
	explorerConfig.ListeningAddress = *listen

	ex, err := explorer.NewExplorer(*explorerConfig, db)
	if err != nil {
		panic(err)
	}

	err = ex.StartExplorer()
	if err != nil {
		panic(err)
	}
}
