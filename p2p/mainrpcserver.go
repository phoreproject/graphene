package p2p

import (
	"context"

	pb "github.com/phoreproject/synapse/pb"
)

type mainRPCServer struct {
}

// NewMainRPCServer creates a new pb.MainRPCServer
func NewMainRPCServer() pb.MainRPCServer {
	return mainRPCServer{}
}

func (server mainRPCServer) Test(ctx context.Context, message *pb.TestMessage) (*pb.TestMessage, error) {
	return &pb.TestMessage{
		Message: message.Message + " === this is response",
	}, nil
}
