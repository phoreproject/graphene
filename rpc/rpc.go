//go:generate protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto

package rpc

import (
	"net"

	"github.com/btcsuite/btcd/chaincfg/chainhash"

	"github.com/phoreproject/synapse/primitives"

	pb "github.com/phoreproject/synapse/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// server is used to implement rpc.BlockchainRPCServer.
type server struct{}

// SubmitBlock submits a block to the network after verifying it
func (s *server) SubmitBlock(ctx context.Context, in *pb.SubmitBlockRequest) (*pb.SubmitBlockResponse, error) {
	b, err := primitives.BlockFromProto(in.Block)
	if err != nil {
		return nil, err
	}
	h := b.Hash()
	return &pb.SubmitBlockResponse{BlockHash: h[:]}, nil
}

func (s *server) GetSlotNumber(ctx context.Context, in *pb.Empty) (*pb.SlotNumberResponse, error) {
	return &pb.SlotNumberResponse{SlotNumber: 0}, nil
}

func (s *server) GetBlockHash(ctx context.Context, in *pb.GetBlockHashRequest) (*pb.GetBlockHashResponse, error) {
	return &pb.GetBlockHashResponse{Hash: chainhash.HashB([]byte("this is a test"))}, nil
}

// Serve serves the RPC server
func Serve(listenAddr string) error {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterBlockchainRPCServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
