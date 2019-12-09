package shardrelayer

import (
	"context"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/relayer/mempool"
	"github.com/phoreproject/synapse/relayer/p2p"
	"github.com/sirupsen/logrus"
)

// ShardRelayer handles state mempool and execution for a single shard.
type ShardRelayer struct {
	mempool     *mempool.ShardMempool
	rpc         pb.ShardRPCClient
	syncManager *p2p.RelayerSyncManager
	shardID     uint64
}

// NewShardRelayer creates a new shard relayer.
func NewShardRelayer(shardID uint64, shardRPC pb.ShardRPCClient, mempool *mempool.ShardMempool, syncManager *p2p.RelayerSyncManager) *ShardRelayer {
	sr := &ShardRelayer{
		mempool:     mempool,
		shardID:     shardID,
		rpc:         shardRPC,
		syncManager: syncManager,
	}

	return sr
}

// GetMempool gets the mempool for this relayer.
func (sr *ShardRelayer) GetMempool() *mempool.ShardMempool {
	return sr.mempool
}

// ListenForActions listens for incoming actions and relays them to the mempool.
func (sr *ShardRelayer) ListenForActions() {
	go func() {
		stream, err := sr.rpc.GetActionStream(context.Background(), &pb.ShardActionStreamRequest{
			ShardID: sr.shardID,
		})
		if err != nil {
			logrus.Error(err)
			return
		}

		for {
			action, err := stream.Recv()
			if err != nil {
				logrus.Error(err)
				return
			}

			err = sr.mempool.AcceptAction(action)
			if err != nil {
				logrus.Error(err)
				return
			}
		}
	}()
}

// GetShardID returns the shard ID of this relayer.
func (sr *ShardRelayer) GetShardID() uint64 {
	return sr.shardID
}
