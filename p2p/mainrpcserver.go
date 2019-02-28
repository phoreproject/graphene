package p2p

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/beacon"
	pb "github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type mainRPCServer struct {
	chain *beacon.Blockchain
}

// NewMainRPCServer creates a new pb.MainRPCServer
func NewMainRPCServer(chain *beacon.Blockchain) pb.MainRPCServer {
	return mainRPCServer{chain: chain}
}

func (s mainRPCServer) Test(ctx context.Context, message *pb.TestMessage) (*pb.TestMessage, error) {
	fmt.Printf("Received test message: %s\n", message.Message)
	return &pb.TestMessage{
		Message: message.Message + " === this is response",
	}, nil
}

// SubmitBlock submits a block to the network after verifying it
func (s mainRPCServer) SubmitBlock(ctx context.Context, in *pb.SubmitBlockRequest) (*pb.SubmitBlockResponse, error) {
	b, err := primitives.BlockFromProto(in.Block)
	if err != nil {
		return nil, err
	}
	err = s.chain.ProcessBlock(b)
	h := b.BlockHeader.ParentRoot // TODO: should be block hash
	return &pb.SubmitBlockResponse{BlockHash: h[:]}, err
}

func (s mainRPCServer) GetSlotNumber(ctx context.Context, in *empty.Empty) (*pb.SlotNumberResponse, error) {
	return &pb.SlotNumberResponse{SlotNumber: uint64(s.chain.Height())}, nil
}

func (s mainRPCServer) GetBlockHash(ctx context.Context, in *pb.GetBlockHashRequest) (*pb.GetBlockHashResponse, error) {
	h, err := s.chain.GetHashByHeight(in.SlotNumber)
	if err != nil {
		return nil, err
	}
	return &pb.GetBlockHashResponse{Hash: h[:]}, nil
}

func (s mainRPCServer) GetSlotAndShardAssignment(ctx context.Context, in *pb.GetSlotAndShardAssignmentRequest) (*pb.SlotAndShardAssignment, error) {
	return nil, nil
	/* to be fixed
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
	*/
}

func (s mainRPCServer) GetValidatorAtIndex(ctx context.Context, in *pb.GetValidatorAtIndexRequest) (*pb.GetValidatorAtIndexResponse, error) {
	return nil, nil

	/* to be fixed
	validator, err := s.chain.GetValidatorAtIndex(in.Index)
	if err != nil {
		return nil, err
	}
	return &pb.GetValidatorAtIndexResponse{Validator: validator.ToProto()}, nil
	*/
}

func (s mainRPCServer) GetCommitteeValidators(ctx context.Context, in *pb.GetCommitteeValidatorsRequest) (*pb.GetCommitteeValidatorsResponse, error) {
	return &pb.GetCommitteeValidatorsResponse{}, nil

	/*
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
				validatorList = append(validatorList, validator.ToProto())
			}

			return &pb.GetCommitteeValidatorsResponse{Validators: validatorList}, nil
	*/
}

// StartMainRPCServe serves the RPC mainRPCServer
func StartMainRPCServe(listenAddr string) error {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	//pb.RegisterBlockchainRPCServer(s, &mainRPCServer{b})
	// Register reflection service on gRPC mainRPCServer.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
