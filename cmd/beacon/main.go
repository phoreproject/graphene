package main

import (
	"flag"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/rpc"
	"github.com/phoreproject/synapse/serialization"

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

	logger.Info("initializing RPC")

	err = rpc.Serve(*rpcConnect, blockchain)
	if err != nil {
		panic(err)
	}
}
