//go:generate protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto

package rpc

import (
	"net"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/beacon"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/beacon/primitives"

	"github.com/phoreproject/synapse/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// server is used to implement rpc.BlockchainRPCServer.
type server struct {
	chain *beacon.Blockchain
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

func (s *server) GetSlotNumber(ctx context.Context, in *empty.Empty) (*pb.SlotNumberResponse, error) {
	return &pb.SlotNumberResponse{SlotNumber: uint64(s.chain.Height())}, nil
}

func (s *server) GetBlockHash(ctx context.Context, in *pb.GetBlockHashRequest) (*pb.GetBlockHashResponse, error) {
	h, err := s.chain.GetNodeByHeight(in.SlotNumber)
	if err != nil {
		return nil, err
	}
	return &pb.GetBlockHashResponse{Hash: h[:]}, nil
}

func (s *server) GetSlotAndShardAssignment(ctx context.Context, in *pb.GetSlotAndShardAssignmentRequest) (*pb.SlotAndShardAssignment, error) {
	shardID, slot, role, err := s.chain.GetSlotAndShardAssignment(in.ValidatorID)
	if err != nil {
		return nil, err
	}

	r := pb.Role_ATTESTER
	if role == beacon.RoleProposer {
		r = pb.Role_PROPOSER
	}

	logrus.WithFields(logrus.Fields{
		"validatorID": in.ValidatorID,
		"shardID":     shardID,
		"slot":        slot,
		"role":        r}).Debug("slot shard assignment")

	return &pb.SlotAndShardAssignment{ShardID: uint32(shardID), Slot: slot, Role: r}, nil
}

func (s *server) GetValidatorAtIndex(ctx context.Context, in *pb.GetValidatorAtIndexRequest) (*pb.GetValidatorAtIndexResponse, error) {
	validator, err := s.chain.GetValidatorAtIndex(in.Index)
	if err != nil {
		return nil, err
	}
	return &pb.GetValidatorAtIndexResponse{Validator: validator.ToProtoResponse(int32(in.Index))}, nil
}

func (s *server) GetCommitteeValidators(ctx context.Context, in *pb.GetCommitteeValidatorsRequest) (*pb.GetCommitteeValidatorsResponse, error) {
	indices, err := s.chain.GetCommitteeValidatorIndices(in.SlotNumber, uint64(in.Shard))
	if err != nil {
		return nil, err
	}

	var validatorList []*pb.ValidatorResponse

	for _, indice := range indices {
		validator, err := s.chain.GetValidatorAtIndex(indice)
		if err != nil {
			return nil, err
		}
		validatorList = append(validatorList, validator.ToProtoResponse(int32(indice)))
	}

	return &pb.GetCommitteeValidatorsResponse{Validators: validatorList}, nil
}

// Serve serves the RPC server
func Serve(listenAddr string, b *beacon.Blockchain) error {
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
