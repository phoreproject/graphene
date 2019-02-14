//go:generate protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto

package rpc

import (
	"net"

	"github.com/phoreproject/prysm/shared/ssz"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/beacon"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/primitives"

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
	if err != nil {
		return nil, err
	}
	h, err := ssz.TreeHash(b)
	if err != nil {
		return nil, err
	}
	return &pb.SubmitBlockResponse{BlockHash: h[:]}, err
}

func (s *server) GetSlotNumber(ctx context.Context, in *empty.Empty) (*pb.SlotNumberResponse, error) {
	return &pb.SlotNumberResponse{SlotNumber: uint64(s.chain.Height())}, nil
}

func (s *server) GetBlockHash(ctx context.Context, in *pb.GetBlockHashRequest) (*pb.GetBlockHashResponse, error) {
	h, err := s.chain.GetHashByHeight(in.SlotNumber)
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
		"role":        r})

	return &pb.SlotAndShardAssignment{ShardID: uint32(shardID), Slot: slot, Role: r}, nil
}

func (s *server) GetValidatorAtIndex(ctx context.Context, in *pb.GetValidatorAtIndexRequest) (*pb.GetValidatorAtIndexResponse, error) {
	validator, err := s.chain.GetValidatorAtIndex(in.Index)
	if err != nil {
		return nil, err
	}
	return &pb.GetValidatorAtIndexResponse{Validator: validator.ToProto()}, nil
}

func (s *server) GetCommitteeValidators(ctx context.Context, in *pb.GetCommitteeValidatorsRequest) (*pb.GetCommitteeValidatorsResponse, error) {
	indices, err := s.chain.GetCommitteeValidatorIndices(s.chain.GetState().Slot, in.SlotNumber, uint64(in.Shard))
	if err != nil {
		return nil, err
	}

	var validatorList []*pb.Validator

	for _, indice := range indices {
		validator, err := s.chain.GetValidatorAtIndex(indice)
		if err != nil {
			return nil, err
		}
		validatorList = append(validatorList, validator.ToProto())
	}

	return &pb.GetCommitteeValidatorsResponse{Validators: validatorList}, nil
}

func (s *server) GetCommitteeValidatorIndices(ctx context.Context, in *pb.GetCommitteeValidatorsRequest) (*pb.GetCommitteeValidatorIndicesResponse, error) {
	indices, err := s.chain.GetCommitteeValidatorIndices(s.chain.GetState().Slot, in.SlotNumber, uint64(in.Shard))
	if err != nil {
		return nil, err
	}

	return &pb.GetCommitteeValidatorIndicesResponse{Validators: indices}, nil
}

func (s *server) GetState(ctx context.Context, in *empty.Empty) (*pb.GetStateResponse, error) {
	state := s.chain.GetState()
	stateProto := state.ToProto()

	return &pb.GetStateResponse{State: stateProto}, nil
}

func (s *server) GetStateRoot(ctx context.Context, in *empty.Empty) (*pb.GetStateRootResponse, error) {
	state := s.chain.GetState()

	root, err := ssz.TreeHash(state)
	if err != nil {
		return nil, err
	}

	return &pb.GetStateRootResponse{StateRoot: root[:]}, nil
}

// GetSlotInformation gets information about the next slot used for attestation
// assignment and generation.
func (s *server) GetSlotInformation(ctx context.Context, in *empty.Empty) (*pb.SlotInformation, error) {
	state := s.chain.GetState()
	config := s.chain.GetConfig()
	beaconBlockHash := s.chain.Tip()
	epochBoundaryRoot, err := s.chain.GetEpochBoundaryHash()
	crosslinks := make([]*pb.Crosslink, len(state.LatestCrosslinks))
	for i := range crosslinks {
		crosslinks[i] = state.LatestCrosslinks[i].ToProto()
	}
	if err != nil {
		return nil, err
	}

	justifiedRoot, err := s.chain.GetHashByHeight(state.JustifiedSlot)
	if err != nil {
		return nil, err
	}

	committeesForNextSlot := state.GetShardCommitteesAtSlot(state.Slot, state.Slot, config)
	committeesForNextSlotProto := make([]*pb.ShardCommittee, len(committeesForNextSlot))
	for i := range committeesForNextSlot {
		committeesForNextSlotProto[i] = committeesForNextSlot[i].ToProto()
	}

	nextProposerIndex := state.GetBeaconProposerIndex(state.Slot, state.Slot, config)

	return &pb.SlotInformation{
		Slot:              state.Slot, // this is the current slot
		BeaconBlockHash:   beaconBlockHash[:],
		EpochBoundaryRoot: epochBoundaryRoot[:],
		LatestCrosslinks:  crosslinks,
		JustifiedSlot:     state.JustifiedSlot,
		JustifiedRoot:     justifiedRoot[:],
		Committees:        committeesForNextSlotProto,
		Proposer:          nextProposerIndex,
	}, nil
}

func (s *server) GetCommitteesForSlot(ctx context.Context, in *pb.GetCommitteesForSlotRequest) (*pb.ShardCommitteesForSlot, error) {
	state := s.chain.GetState()

	sc := state.GetShardCommitteesAtSlot(state.Slot, in.Slot, s.chain.GetConfig())
	scProto := make([]*pb.ShardCommittee, len(sc))
	for i := range sc {
		scProto[i] = sc[i].ToProto()
	}

	return &pb.ShardCommitteesForSlot{
		Committees: scProto,
	}, nil
}

func (s *server) GetForkData(ctx context.Context, in *empty.Empty) (*pb.ForkData, error) {
	state := s.chain.GetState()
	return state.ForkData.ToProto(), nil
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
