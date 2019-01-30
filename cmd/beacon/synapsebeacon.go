package main

import (
	"context"
	"flag"

	"github.com/golang/protobuf/proto"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/beacon/primitives"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/rpc"
	"github.com/phoreproject/synapse/serialization"
	"google.golang.org/grpc"

	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/db"

	logger "github.com/sirupsen/logrus"
)

const clientVersion = "0.0.1"

func main() {
	p2pConnect := flag.String("p2pconnect", "127.0.0.1:11783", "host and port for P2P rpc connection")
	rpcConnect := flag.String("rpclisten", "127.0.0.1:11782", "host and port for RPC server to listen on")
	flag.Parse()

	logger.WithField("version", clientVersion).Info("initializing client")

	logger.Info("initializing database")
	database := db.NewInMemoryDB()
	c := beacon.MainNetConfig

	logger.Info("initializing blockchain")

	validators := []beacon.InitialValidatorEntry{}

	randaoCommitment := chainhash.HashH([]byte("test"))

	for i := 0; i <= c.EpochLength*(c.MinCommitteeSize*2); i++ {
		validators = append(validators, beacon.InitialValidatorEntry{
			PubKey:                bls.PublicKey{},
			ProofOfPossession:     bls.Signature{},
			WithdrawalShard:       1,
			WithdrawalCredentials: serialization.Address{},
			RandaoCommitment:      randaoCommitment,
		})
	}

	logger.WithField("numValidators", len(validators)).Info("initializing blockchain with validators")

	blockchain, err := beacon.NewBlockchainWithInitialValidators(database, &c, validators)
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

			var blockProto *pb.Block

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

	err = rpc.Serve(*rpcConnect, blockchain)
	if err != nil {
		panic(err)
	}
}
