package main

import (
	"encoding/binary"
	"flag"
	"os"

	"github.com/phoreproject/synapse/p2p"

	"github.com/phoreproject/synapse/beacon/app"

	"github.com/phoreproject/synapse/beacon"

	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
)

const clientVersion = "0.0.1"

func main() {
	rpcConnect := flag.String("rpclisten", "127.0.0.1:11782", "host and port for RPC server to listen on")
	genesisTime := flag.Uint64("genesistime", 0, "beacon chain genesis time")
	initialpubkeys := flag.String("initialpubkeys", "testnet.pubs", "file of pub keys for initial validators")
	resync := flag.Bool("resync", false, "resyncs the blockchain if this is set")
	datadir := flag.String("datadir", "", "location to store blockchain data")

	// P2P
	initialConnections := flag.String("connect", "", "comma separated multiaddrs")
	listen := flag.String("listen", "/ip4/0.0.0.0/tcp/11781", "specifies the address to listen on")

	// Logging
	level := flag.String("level", "info", "log level")
	flag.Parse()

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

	appConfig := app.NewConfig()
	appConfig.ListeningAddress = *listen
	appConfig.RPCAddress = *rpcConnect
	appConfig.DiscoveryOptions.PeerAddresses = initialPeers
	appConfig.DataDirectory = *datadir

	appConfig.Resync = *resync
	if *genesisTime != 0 {
		appConfig.GenesisTime = *genesisTime
	}

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

	appConfig.InitialValidatorList = make([]beacon.InitialValidatorEntry, length)

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
		appConfig.InitialValidatorList[validatorID] = iv
	}

	a := app.NewBeaconApp(appConfig)
	err = a.Run()
	if err != nil {
		panic(err)
	}
}
