package main

import (
	"context"
	"flag"

	"github.com/golang/protobuf/proto"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/rpc"
	"github.com/phoreproject/synapse/serialization"
	"google.golang.org/grpc"

	"github.com/phoreproject/synapse/blockchain"
	"github.com/phoreproject/synapse/db"

	logger "github.com/inconshreveable/log15"
)

const clientVersion = "0.0.1"

func main() {
	p2pConnect := flag.String("p2pconnect", "127.0.0.1:11783", "host and port for P2P rpc connection")
	rpcConnect := flag.String("rpclisten", "127.0.0.1:11782", "host and port for RPC server to listen on")
	flag.Parse()

	logger.Info("initializing client", "version", clientVersion)

	logger.Info("initializing database")
	database := db.NewInMemoryDB()
	c := blockchain.MainNetConfig

	logger.Info("initializing blockchain")

	validators := []blockchain.InitialValidatorEntry{}

	randaoCommitment := chainhash.HashH([]byte("test"))

	for i := 0; i <= c.CycleLength*(c.MinCommitteeSize*2); i++ {
		validators = append(validators, blockchain.InitialValidatorEntry{
			PubKey:            bls.PublicKey{},
			ProofOfPossession: bls.Signature{},
			WithdrawalShard:   1,
			WithdrawalAddress: serialization.Address{},
			RandaoCommitment:  randaoCommitment,
		})
	}

	logger.Info("initializing blockchain with validators", "numValidators", len(validators))

	blockchain, err := blockchain.NewBlockchainWithInitialValidators(database, &c, validators)
	if err != nil {
		panic(err)
	}

	logger.Info("connecting to p2p RPC")
	conn, err := grpc.Dial(*p2pConnect, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	p2p := pb.NewP2PRPCClient(conn)
	_, err = p2p.Connect(context.Background(), &pb.InitialPeers{})

	if err != nil {
		panic(err)
	}

	listener, err := p2p.ListenForMessages(context.Background(), &pb.SubscriptionRequest{Topic: "block"})
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

			err = proto.Unmarshal(msg.Response, blockProto)
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
