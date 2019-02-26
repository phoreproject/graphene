//go:generate protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto

package rpc

import (
	"errors"
	"net"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/beacon"

	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// server is used to implement rpc.BlockchainRPCServer.
type server struct {
	chain *beacon.Blockchain
	p2p   pb.P2PRPCClient
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

	data, err := proto.Marshal(b.ToProto())
	if err != nil {
		return nil, err
	}

	_, err = s.p2p.Broadcast(context.Background(), &pb.MessageAndTopic{
		Topic: "block",
		Data:  data,
	})
	if err != nil {
		return nil, err
	}
	return &pb.SubmitBlockResponse{BlockHash: h[:]}, err
}

func (s *server) GetSlotNumber(ctx context.Context, in *empty.Empty) (*pb.SlotNumberResponse, error) {
	state := s.chain.GetState()
	config := s.chain.GetConfig()
	genesisTime := state.GenesisTime
	timePerSlot := config.SlotDuration
	currentTime := time.Now().Unix()
	currentSlot := (currentTime-int64(genesisTime))/int64(timePerSlot) - 1
	if currentSlot < 0 {
		currentSlot = 0
	}
	block := s.chain.Tip()
	return &pb.SlotNumberResponse{SlotNumber: uint64(currentSlot), BlockHash: block[:]}, nil
}

func (s *server) GetBlockHash(ctx context.Context, in *pb.GetBlockHashRequest) (*pb.GetBlockHashResponse, error) {
	h, err := s.chain.GetHashByHeight(in.SlotNumber)
	if err != nil {
		return nil, err
	}
	return &pb.GetBlockHashResponse{Hash: h[:]}, nil
}

func (s *server) GetLastBlockHash(ctx context.Context, in *empty.Empty) (*pb.GetBlockHashResponse, error) {
	h := s.chain.Tip()
	return &pb.GetBlockHashResponse{
		Hash: h[:],
	}, nil
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
func (s *server) GetEpochInformation(ctx context.Context, in *empty.Empty) (*pb.EpochInformation, error) {
	state := s.chain.GetState()
	config := s.chain.GetConfig()
	genesisTime := state.GenesisTime
	timePerSlot := config.SlotDuration
	currentTime := time.Now().Unix()
	currentSlot := (currentTime - int64(genesisTime)) / int64(timePerSlot)

	if currentSlot < 0 {
		return &pb.EpochInformation{
			Slot: -1,
		}, nil
	}

	if currentSlot > int64(state.Slot) {
		err := s.chain.UpdateStateIfNeeded(uint64(currentSlot))
		if err != nil {
			return nil, err
		}

		state = s.chain.GetState()
	}

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

	earliestSlot := int64(state.Slot) - int64(state.Slot%config.EpochLength) - int64(config.EpochLength)

	slots := make([]*pb.SlotInformation, len(state.ShardAndCommitteeForSlots))
	for s := range slots {
		shardAndCommittees := make([]*pb.ShardCommittee, len(state.ShardAndCommitteeForSlots[s]))
		for i := range shardAndCommittees {
			shardAndCommittees[i] = state.ShardAndCommitteeForSlots[s][i].ToProto()
		}
		slots[s] = &pb.SlotInformation{
			Slot:       earliestSlot + int64(s) + 1,
			Committees: shardAndCommittees,
			ProposeAt:  uint64(earliestSlot+int64(s)+1)*uint64(config.SlotDuration) + state.GenesisTime,
		}
	}

	return &pb.EpochInformation{
		Slots:             slots,
		Slot:              int64(state.Slot) - int64(state.Slot%config.EpochLength),
		EpochBoundaryRoot: epochBoundaryRoot[:],
		LatestCrosslinks:  crosslinks,
		JustifiedSlot:     state.JustifiedSlot,
		JustifiedHash:     justifiedRoot[:],
	}, nil
}

func (s *server) GetCommitteesForSlot(ctx context.Context, in *pb.GetCommitteesForSlotRequest) (*pb.ShardCommitteesForSlot, error) {
	state := s.chain.GetState()

	sc, err := state.GetShardCommitteesAtSlot(state.Slot, in.Slot, s.chain.GetConfig())
	if err != nil {
		return nil, err
	}
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

func (s *server) GetProposerSlots(ctx context.Context, in *pb.GetProposerSlotsRequest) (*pb.GetProposerSlotsResponse, error) {
	state := s.chain.GetState()
	if in.ValidatorID >= uint32(len(state.ValidatorRegistry)) {
		return nil, errors.New("validator ID out of range")
	}

	return &pb.GetProposerSlotsResponse{
		ProposerSlots: state.ValidatorRegistry[in.ValidatorID].ProposerSlots,
	}, nil
}

// Serve serves the RPC server
func Serve(listenAddr string, b *beacon.Blockchain, p2p pb.P2PRPCClient) error {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterBlockchainRPCServer(s, &server{b, p2p})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
