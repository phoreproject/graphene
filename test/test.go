package main

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/pb"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("127.0.0.1:11883", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	client0 := pb.NewP2PRPCClient(conn)

	conn1, err := grpc.Dial("127.0.0.1:11783", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	client1 := pb.NewP2PRPCClient(conn1)

	peers0, err := client0.GetPeers(context.Background(), &empty.Empty{})
	if err != nil {
		panic(err)
	}
	fmt.Print("Peer 0 Peers: ")
	for _, p := range peers0.Peers {
		fmt.Printf("%s, ", p.Address)
	}
	fmt.Println()
	peers1, err := client1.GetPeers(context.Background(), &empty.Empty{})
	if err != nil {
		panic(err)
	}
	fmt.Print("Peer 1 Peers: ")
	for _, p := range peers1.Peers {
		fmt.Printf("%s, ", p.Address)
	}
	fmt.Println()

	s, err := client0.Subscribe(context.Background(), &pb.SubscriptionRequest{Topic: "cows"})
	if err != nil {
		panic(err)
	}

	stream, err := client0.ListenForMessages(context.Background(), s)
	if err != nil {
		panic(err)
	}

	doneChan := make(chan struct{})

	go func() {
		msg, err := stream.Recv()
		if err != nil {
			panic(err)
		}

		fmt.Printf("peer 0 returned message from topic cow: %s\n", msg)

		doneChan <- struct{}{}
	}()

	fmt.Println("peer 1 sending message to topic cow: I'm a cow")

	_, err = client1.Broadcast(context.Background(), &pb.MessageAndTopic{Data: []byte("I'm a cow"), Topic: "cows"})
	if err != nil {
		panic(err)
	}

	<-doneChan
}
