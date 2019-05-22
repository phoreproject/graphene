package main

import (
	"encoding/binary"
	"flag"
	"os"

	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/utils"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/phoreproject/synapse/explorer"

	"github.com/sirupsen/logrus"
)

func main() {
	genesisTime := flag.Uint64("genesistime", 0, "beacon chain genesis time")
	initialpubkeys := flag.String("initialpubkeys", "testnet.pubs", "file of pub keys for initial validators")
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
	f, err := os.Open(*initialpubkeys)
	if err != nil {
		panic(err)
	}

	var lengthBytes [4]byte

	_, err = f.Read(lengthBytes[:])
	if err != nil {
		panic(err)
	}

	length := binary.BigEndian.Uint32(lengthBytes[:])

	initialValidatorList := make([]beacon.InitialValidatorEntry, length)

	for i := uint32(0); i < length; i++ {
		var validatorIDBytes [4]byte
		n, err := f.Read(validatorIDBytes[:])
		if err != nil {
			panic(err)
		}
		if n != 4 {
			panic("unexpected end of pubkey file")
		}

		validatorID := binary.BigEndian.Uint32(validatorIDBytes[:])

		var iv beacon.InitialValidatorEntry
		err = binary.Read(f, binary.BigEndian, &iv)
		if err != nil {
			panic(err)
		}
		initialValidatorList[validatorID] = iv
	}

	initialPeers, err := p2p.ParseInitialConnections(*initialConnections)
	if err != nil {
		panic(err)
	}

	options := p2p.NewDiscoveryOptions()
	options.PeerAddresses = initialPeers

	explorerConfig := explorer.Config{
		GenesisTime:          *genesisTime,
		DataDirectory:        *datadir,
		Resync:               *resync,
		InitialValidatorList: initialValidatorList,
		NetworkConfig:        &config.MainNetConfig,
		ListeningAddress:     *listen,
		DiscoveryOptions:     options,
	}

	ex, err := explorer.NewExplorer(explorerConfig, db)
	if err != nil {
		panic(err)
	}

	err = ex.StartExplorer()
	if err != nil {
		panic(err)
	}
}
