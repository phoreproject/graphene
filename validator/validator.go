package main

import (
	"context"
	"flag"

	"google.golang.org/grpc"

	"github.com/inconshreveable/log15"
	"github.com/phoreproject/synapse/pb"
)

func main() {
	log15.Info("Starting validator")
	beaconHost := flag.String("beaconhost", ":11782", "the address to connect to the beacon node")
	flag.Parse()

	conn, err := grpc.Dial(*beaconHost, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	rpcClient := pb.NewBlockchainRPCClient(conn)
	slot, err := rpcClient.GetSlotNumber(context.Background(), &pb.Empty{})
	if err != nil {
		panic(err)
	}

	log15.Info("got slot number", "slot", slot.SlotNumber)
}
