package main

import (
	"flag"

	"github.com/phoreproject/synapse/explorer"
	"google.golang.org/grpc"
)

func main() {
	beaconHost := flag.String("beaconhost", ":11782", "the address to connect to the beacon node")
	flag.Parse()

	blockchainConn, err := grpc.Dial(*beaconHost, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	ex := explorer.NewExplorer(blockchainConn)
	err = ex.StartExplorer()
	if err != nil {
		panic(err)
	}
}
