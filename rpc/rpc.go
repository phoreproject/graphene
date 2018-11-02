//go:generate protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto

package rpc

import (
	"net"

	"github.com/phoreproject/synapse/blockchain"

	"github.com/phoreproject/synapse/primitives"

	pb "github.com/phoreproject/synapse/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// server is used to implement rpc.BlockchainRPCServer.
type server struct {
	chain *blockchain.Blockchain
}

// SubmitBlock submits a block to the network after verifying it
func (s *server) SubmitBlock(ctx context.Context, in *pb.SubmitBlockRequest) (*pb.SubmitBlockResponse, error) {
	b, err := primitives.BlockFromProto(in.Block)
	if err != nil {
		return nil, err
	}
	err = s.chain.ProcessBlock(b)
	h := b.Hash()
	return &pb.SubmitBlockResponse{BlockHash: h[:]}, err
}

func (s *server) GetSlotNumber(ctx context.Context, in *pb.Empty) (*pb.SlotNumberResponse, error) {
	return &pb.SlotNumberResponse{SlotNumber: uint64(s.chain.Height())}, nil
}

func (s *server) GetBlockHash(ctx context.Context, in *pb.GetBlockHashRequest) (*pb.GetBlockHashResponse, error) {
	h, err := s.chain.GetNodeByHeight(in.SlotNumber)
	if err != nil {
		return nil, err
	}
	return &pb.GetBlockHashResponse{Hash: h[:]}, nil
}

// Serve serves the RPC server
func Serve(listenAddr string, b *blockchain.Blockchain) error {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterBlockchainRPCServer(s, &server{b})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
