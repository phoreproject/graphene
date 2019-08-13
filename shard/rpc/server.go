package rpc

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/pb"
)

// ShardRPCServer handles incoming commands for the shard module.
type ShardRPCServer struct {
}

// SubscribeToShard instructs the shard module to subscribe to a specific shard and start downloading blocks. This is a
// no-op until we implement the P2P network.
func (ShardRPCServer) SubscribeToShard(context.Context, *pb.ShardSubscription) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

// UnsubscribeFromShard instructs the shard module to unsubscribe from a specific shard including disconnecting from
// unnecessary peers and clearing the database of any unnecessary state. No-op until we implement the P2P network.
func (ShardRPCServer) UnsubscribeFromShard(context.Context, *pb.ShardSubscription) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

// GetBlockHashAtSlot gets the block hash at a specific slot on a specific shard.
func (ShardRPCServer) GetBlockHashAtSlot(context.Context, *pb.SlotRequest) (*pb.BlockHashResponse, error) {
	return nil, nil
}

// GenerateBlockTemplate generates a block template using transactions and witnesses for a certain shard at a certain
// slot.
func (ShardRPCServer) GenerateBlockTemplate(context.Context, *pb.SlotRequest) (*pb.ShardBlock, error) {
	panic("implement me")
}

// SubmitBlock submits a block to the shard chain.
func (ShardRPCServer) SubmitBlock(context.Context, *pb.ShardBlock) (*empty.Empty, error) {
	panic("implement me")
}

var _ pb.ShardRPCServer = &ShardRPCServer{}
