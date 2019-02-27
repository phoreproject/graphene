package main

import (
	"context"
	"encoding/binary"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/rpc"
	"google.golang.org/grpc"

	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/db"

	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
)

const clientVersion = "0.0.1"

func main() {
	logrus.SetLevel(logrus.DebugLevel)

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
