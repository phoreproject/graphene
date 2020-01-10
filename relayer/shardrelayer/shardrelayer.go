package shardrelayer

import (
	"context"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
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

// GetStateKey gets a key from state.
func (sr *ShardRelayer) GetStateKey(key chainhash.Hash) (chainhash.Hash, error) {
	var val chainhash.Hash
	err := sr.mempool.GetTipState().View(func(tx csmt.TreeDatabaseTransaction) error {
		v, err := tx.Get(key)
		if err != nil {
			return err
		}
		val = *v
		return nil
	})
	return val, err
}

// GetStateKeys gets keys from state.
func (sr *ShardRelayer) GetStateKeys(keys []chainhash.Hash) ([]chainhash.Hash, error) {
	vals := make([]chainhash.Hash, len(keys))
	err := sr.mempool.GetTipState().View(func(tx csmt.TreeDatabaseTransaction) error {
		for i, k := range keys {
			v, err := tx.Get(k)
			if err != nil {
				return err
			}
			vals[i] = *v
		}
		return nil
	})
	return vals, err
}

