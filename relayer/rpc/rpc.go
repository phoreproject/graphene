package rpc

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

type server struct {

}

func (server) SubmitTransaction(context.Context, *pb.ShardTransaction) (*empty.Empty, error) {
	return nil, nil
}

// Serve serves the RPC server
func Serve(proto string, listenAddr string) error {
	lis, err := net.Listen(proto, listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterRelayerRPCServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
