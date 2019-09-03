package rpc

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/chain"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

// ShardRPCServer handles incoming commands for the shard module.
type ShardRPCServer struct {
	sm *chain.ShardMux
}

// SubscribeToShard instructs the shard module to subscribe to a specific shard and start downloading blocks. This is a
// no-op until we implement the P2P network.
func (s *ShardRPCServer) SubscribeToShard(ctx context.Context, req *pb.ShardSubscribeRequest) (*empty.Empty, error) {
	blockHash, err := chainhash.NewHash(req.BlockHash)
	if err != nil {
		return nil, err
	}

	logrus.Infof("subscribing to shard %d with crosslink at %d and latest block hash %s", req.ShardID,
		req.CrosslinkSlot, blockHash)

	if !s.sm.IsManaging(req.ShardID) {
		s.sm.StartManaging(req.ShardID, chain.ShardChainInitializationParameters{
			RootBlockHash: *blockHash,
			RootSlot:      req.CrosslinkSlot,
		})
	}

	return &empty.Empty{}, nil
}

// UnsubscribeFromShard instructs the shard module to unsubscribe from a specific shard including disconnecting from
// unnecessary peers and clearing the database of any unnecessary state. No-op until we implement the P2P network.
func (s *ShardRPCServer) UnsubscribeFromShard(ctx context.Context, req *pb.ShardUnsubscribeRequest) (*empty.Empty, error) {
	logrus.Infof("unsubscribing to shard %d", req.ShardID)

	s.sm.StopManaging(req.ShardID)

	return &empty.Empty{}, nil
}

// GetBlockHashAtSlot gets the block hash at a specific slot on a specific shard.
func (s *ShardRPCServer) GetBlockHashAtSlot(ctx context.Context, req *pb.SlotRequest) (*pb.BlockHashResponse, error) {
	if req.Slot == 0 {
		genesisBlock := primitives.GetGenesisBlockForShard(req.Shard)

		genesisHash, err := ssz.HashTreeRoot(genesisBlock)
		if err != nil {
			return nil, err
		}

		return &pb.BlockHashResponse{
			BlockHash: genesisHash[:],
		}, nil
	}

	manager, err := s.sm.GetManager(req.Shard)
	if err != nil {
		return nil, err
	}

	blockNode, err := manager.Chain.GetNodeBySlot(req.Slot)
	if err != nil {
		return nil, err
	}

	return &pb.BlockHashResponse{
		BlockHash: blockNode.BlockHash[:],
	}, nil
}

// GenerateBlockTemplate generates a block template using transactions and witnesses for a certain shard at a certain
// slot.
func (s *ShardRPCServer) GenerateBlockTemplate(ctx context.Context, req *pb.BlockGenerationRequest) (*pb.ShardBlock, error) {
	logrus.WithFields(logrus.Fields{
		"shard": req.Shard,
		"slot":  req.Slot,
	}).Debug("generating block template for signing")

	finalizedBeaconHash, err := chainhash.NewHash(req.FinalizedBeaconHash)
	if err != nil {
		return nil, err
	}

	manager, err := s.sm.GetManager(req.Shard)
	if err != nil {
		return nil, err
	}

	tipNode, err := manager.Chain.Tip()
	if err != nil {
		return nil, err
	}

	// for now, block is empty, but we'll fill this in eventually
	block := &primitives.ShardBlock{
		Header: primitives.ShardBlockHeader{
			PreviousBlockHash:   tipNode.BlockHash,
			Slot:                req.Slot,
			Signature:           [48]byte{},
			StateRoot:           chainhash.Hash{},
			TransactionRoot:     chainhash.Hash{},
			FinalizedBeaconHash: *finalizedBeaconHash,
		},
		Body: primitives.ShardBlockBody{
			Transactions: nil,
		},
	}

	blockHash, err := ssz.HashTreeRoot(block)
	if err != nil {
		return nil, err
	}

	logrus.Infof("generated shard block skeleton for shard %d with block hash %x and previous block hash %s",
		req.Shard, blockHash, block.Header.PreviousBlockHash)

	return block.ToProto(), nil
}

// SubmitBlock submits a block to the shard chain.
func (s *ShardRPCServer) SubmitBlock(ctx context.Context, req *pb.ShardBlockSubmission) (*empty.Empty, error) {
	block, err := primitives.ShardBlockFromProto(req.Block)
	if err != nil {
		return nil, err
	}

	blockHash, err := ssz.HashTreeRoot(block)
	if err != nil {
		return nil, err
	}

	logrus.Infof("submitting shard block for shard %d with block hash %x",
		req.Shard, blockHash)

	manager, err := s.sm.GetManager(req.Shard)
	if err != nil {
		return nil, err
	}

	err = manager.SubmitBlock(*block)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

var _ pb.ShardRPCServer = &ShardRPCServer{}

// Serve serves the RPC server
func Serve(proto string, listenAddr string, mux *chain.ShardMux) error {
	lis, err := net.Listen(proto, listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterShardRPCServer(s, &ShardRPCServer{mux})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
