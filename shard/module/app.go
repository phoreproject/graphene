package module

import (
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/shard/chain"
	"github.com/phoreproject/synapse/shard/config"
	"github.com/phoreproject/synapse/shard/rpc"
	"github.com/sirupsen/logrus"
)

const shardExecutionVersion = "0.0.1"

// ShardApp runs the shard microservice which handles state execution and fork choice on shards.
type ShardApp struct {
	Config config.ShardConfig
	RPC    rpc.ShardRPCServer
}

// NewShardApp creates a new shard app given a config.
func NewShardApp(c config.ShardConfig) *ShardApp {
	return &ShardApp{Config: c}
}

// Run runs the shard app.
func (s *ShardApp) Run() error {
	logrus.Info("starting shard version", shardExecutionVersion)

	logrus.Info("initializing shard manager")

	client := pb.NewBlockchainRPCClient(s.Config.BeaconConn)

	mux := chain.NewShardMux(client)

	logrus.Infof("starting RPC server on %s with protocol %s", s.Config.RPCAddress, s.Config.RPCProtocol)

	err := rpc.Serve(s.Config.RPCProtocol, s.Config.RPCAddress, mux)

	return err
}
