package app

import (
	"github.com/phoreproject/synapse/shard/rpc"
	"github.com/sirupsen/logrus"
)

const shardExecutionVersion = "0.0.1"

// ShardApp runs the shard microservice which handles state execution and fork choice on shards.
type ShardApp struct {
	Config ShardConfig
	RPC    rpc.ShardRPCServer
}

// NewShardApp creates a new shard app given a config.
func NewShardApp(c ShardConfig) *ShardApp {
	return &ShardApp{Config: c}
}

// Run runs the shard app.
func (s *ShardApp) Run() {
	logrus.Info("starting shard version", shardExecutionVersion)

}
