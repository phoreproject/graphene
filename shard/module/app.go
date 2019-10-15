package module

import (
	"fmt"

	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/shard/chain"
	"github.com/phoreproject/synapse/shard/config"
	"github.com/phoreproject/synapse/shard/rpc"
	"github.com/phoreproject/synapse/utils"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const shardExecutionVersion = "0.0.1"

// ShardApp runs the shard microservice which handles state execution and fork choice on shards.
type ShardApp struct {
	Config config.ShardConfig
	RPC    rpc.ShardRPCServer
	Mux    *chain.ShardMux
}

// NewShardApp creates a new shard app given a config.
func NewShardApp(options config.Options) (*ShardApp, error) {
	beaconAddr, err := utils.MultiaddrStringToDialString(options.BeaconRPC)
	if err != nil {
		return nil, err
	}

	cc, err := grpc.Dial(beaconAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("could not connect to beacon host at %s with error: %s", options.BeaconRPC, err)
	}

	ma, err := multiaddr.NewMultiaddr(options.RPCListen)
	if err != nil {
		return nil, err
	}

	addr, err := manet.ToNetAddr(ma)
	if err != nil {
		return nil, err
	}

	c := config.ShardConfig{
		BeaconConn:  cc,
		RPCProtocol: addr.Network(),
		RPCAddress:  addr.String(),
	}

	return &ShardApp{Config: c}, nil
}

// Run runs the shard app.
func (s *ShardApp) Run() error {
	logger.Info("starting shard version", shardExecutionVersion)

	logger.Info("initializing shard manager")

	client := pb.NewBlockchainRPCClient(s.Config.BeaconConn)

	s.Mux = chain.NewShardMux(client)

	logger.Infof("starting RPC server on %s with protocol %s", s.Config.RPCAddress, s.Config.RPCProtocol)

	err := rpc.Serve(s.Config.RPCProtocol, s.Config.RPCAddress, s.Mux)

	return err
}

// Exit exits the module.
func (s *ShardApp) Exit() {}
