package main

import (
	"encoding/binary"
	"flag"
	"os"

	"github.com/phoreproject/synapse/beacon/app"

	"github.com/phoreproject/synapse/beacon"

	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
)

const clientVersion = "0.0.1"

/*
func main() {
	logrus.SetLevel(logrus.InfoLevel)

	p2pConnect := flag.String("p2pconnect", "127.0.0.1:11783", "host and port for P2P rpc connection")
	rpcConnect := flag.String("rpclisten", "127.0.0.1:11782", "host and port for RPC server to listen on")
	genesisTime := flag.Uint64("genesistime", 0, "beacon chain genesis time")
	initialpubkeys := flag.String("initialpubkeys", "testnet.pubs", "file of pub keys for initial validators")
	flag.Parse()

	dir, err := config.GetBaseDirectory(true)
	if err != nil {
		panic(err)
	}

	dbDir := filepath.Join(dir, "db")
	dbValuesDir := filepath.Join(dir, "dbv")

	os.MkdirAll(dbDir, 0777)
	os.MkdirAll(dbValuesDir, 0777)

	logger.WithField("version", clientVersion).Info("initializing client")

	logger.Info("initializing database")
	database := db.NewBadgerDB(dbDir, dbValuesDir)

	defer database.Close()

	c := config.MainNetConfig

	logger.Info("initializing blockchain")

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

	validators := make([]beacon.InitialValidatorEntry, length)

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

		if validatorID != i {
			panic("malformatted pubkey file")
		}

		var iv beacon.InitialValidatorEntry
		err = binary.Read(f, binary.BigEndian, &iv)
		if err != nil {
			panic(err)
		}
		validators[i] = iv
	}

	if *genesisTime == 0 {
		*genesisTime = uint64(time.Now().Unix())
	}

	logger.WithField("numValidators", len(validators)).WithField("genesisTime", *genesisTime).Info("initializing blockchain with validators")
	blockchain, err := beacon.NewBlockchainWithInitialValidators(database, &c, validators, true, *genesisTime)
	if err != nil {
		panic(err)
	}

	logger.Info("connecting to p2p RPC")
	conn, err := grpc.Dial(*p2pConnect, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	p2p := pb.NewP2PRPCClient(conn)

	sub, err := p2p.Subscribe(context.Background(), &pb.SubscriptionRequest{Topic: "block"})
	if err != nil {
		panic(err)
	}

	listener, err := p2p.ListenForMessages(context.Background(), sub)
	if err != nil {
		panic(err)
	}
	go func() {
		newBlocks := make(chan primitives.Block)
		go func() {
			err := blockchain.HandleNewBlocks(newBlocks)
			if err != nil {
				panic(err)
			}
		}()

		for {
			msg, err := listener.Recv()
			if err != nil {
				panic(err)
			}

			blockProto := new(pb.Block)

			err = proto.Unmarshal(msg.Data, blockProto)
			if err != nil {
				continue
			}

			block, err := primitives.BlockFromProto(blockProto)
			if err != nil {
				continue
			}

			newBlocks <- *block
		}
	}()

	logger.Info("initializing RPC")

	go func() {
		err = rpc.Serve(*rpcConnect, blockchain, p2p)
		if err != nil {
			panic(err)
		}
	}()

	signalHandler := make(chan os.Signal, 1)
	signal.Notify(signalHandler, os.Interrupt)
	<-signalHandler

	logger.Info("exiting")
}
*/

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	p2pConnect := flag.String("p2pconnect", "127.0.0.1:11783", "host and port for P2P rpc connection")
	rpcConnect := flag.String("rpclisten", "127.0.0.1:11782", "host and port for RPC server to listen on")
	genesisTime := flag.Uint64("genesistime", 0, "beacon chain genesis time")
	fakeValidatorKeyStore := flag.String("fakevalidatorkeys", "keypairs.bin", "key file of fake validators")
	flag.Parse()

	logger.WithField("version", clientVersion).Info("initializing client")

	appConfig := app.NewConfig()
	appConfig.P2PAddress = *p2pConnect
	appConfig.RPCAddress = *rpcConnect
	if *genesisTime != 0 {
		appConfig.GenesisTime = *genesisTime
	}

	if fakeValidatorKeyStore == nil {
		// we should load this data from chain config file
		// for i := uint64(0); i <= c.EpochLength*(uint64(c.TargetCommitteeSize)*2); i++ {
		// 	priv := keystore.GetKeyForValidator(uint32(i))
		// 	pub := priv.DerivePublicKey()
		// 	hashPub, err := ssz.TreeHash(pub.Serialize())
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	proofOfPossession, err := bls.Sign(priv, hashPub[:], bls.DomainDeposit)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	validators = append(validators, beacon.InitialValidatorEntry{
		// 		PubKey:                pub.Serialize(),
		// 		ProofOfPossession:     proofOfPossession.Serialize(),
		// 		WithdrawalShard:       1,
		// 		WithdrawalCredentials: chainhash.Hash{},
		// 		DepositSize:           c.MaxDeposit * config.UnitInCoin,
		// 	})
		// }
	} else {
		// we should load the keys from the validator keystore
		f, err := os.Open(*fakeValidatorKeyStore)
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
			var iv beacon.InitialValidatorEntryAndPrivateKey
			binary.Read(f, binary.BigEndian, &iv)
			appConfig.InitialValidatorList[i] = iv.Entry
		}
	}

	app := app.NewBeaconApp(appConfig)
	app.Run()
}
