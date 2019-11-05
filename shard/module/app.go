package module

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/chain"
	"github.com/phoreproject/synapse/shard/config"
	beaconconfig "github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/shard/rpc"
	"github.com/phoreproject/synapse/utils"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const shardExecutionVersion = "0.0.1"

// ShardApp runs the shard microservice which handles state execution and fork choice on shards.
type ShardApp struct {
	Config    config.ShardConfig
	RPC       rpc.ShardRPCServer
	beaconRPC pb.BlockchainRPCClient
	shardMux  *chain.ShardMux
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
		TrackShards: options.TrackShards,
	}

	a := &ShardApp{Config: c}

	a.beaconRPC = pb.NewBlockchainRPCClient(a.Config.BeaconConn)

	a.shardMux = chain.NewShardMux(a.beaconRPC)

	genesisTime, err := a.beaconRPC.GetGenesisTime(context.Background(), &empty.Empty{})

	shards, err := utils.ParseRanges(a.Config.TrackShards)
	if err != nil {
		return nil, err
	}

	for _, shard := range shards {
		shardGenesis := primitives.GetGenesisBlockForShard(uint64(shard))

		shardGenesisHash, _ := ssz.HashTreeRoot(shardGenesis)

		a.shardMux.StartManaging(uint64(shard), chain.ShardChainInitializationParameters{
			RootBlockHash: shardGenesisHash,
			RootSlot:      0,
			GenesisTime:   genesisTime.GenesisTime,
		})
	}

	networkConfig, found := beaconconfig.NetworkIDs[options.NetworkID]
	if !found {
		return nil, fmt.Errorf("could not find network config %s", options.NetworkID)
	}

	go func() {
		err := rpc.Serve(a.Config.RPCProtocol, a.Config.RPCAddress, a.shardMux, &networkConfig)
		if err != nil {
			logrus.Error(err)
		}
	}()

	return a, nil
}

// Run runs the shard app.
func (s *ShardApp) Run() error {
	logrus.Info("starting shard version", shardExecutionVersion)

	logrus.Info("initializing shard manager")

	logrus.Infof("starting RPC server on %s with protocol %s", s.Config.RPCAddress, s.Config.RPCProtocol)

	<- make(chan struct{})

	return nil
}

// Exit exits the module.
func (s *ShardApp) Exit() {}
