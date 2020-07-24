package module

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	beaconconfig "github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/chain"
	"github.com/phoreproject/synapse/shard/config"
	"github.com/phoreproject/synapse/shard/rpc"
	"github.com/phoreproject/synapse/ssz"
	"github.com/phoreproject/synapse/utils"
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
	hostnode  *p2p.HostNode
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
		P2PListen:   options.P2PListen,
	}

	a := &ShardApp{Config: c}

	a.beaconRPC = pb.NewBlockchainRPCClient(a.Config.BeaconConn)

	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic(err)
	}

	p2pAddr, err := multiaddr.NewMultiaddr(c.P2PListen)
	if err != nil {
		return nil, err
	}

	hn, err := p2p.NewHostNode(context.TODO(), p2p.HostNodeOptions{
		ListenAddresses: []multiaddr.Multiaddr{p2pAddr},
		PrivateKey:      priv,
		ConnManagerOptions: p2p.ConnectionManagerOptions{
			BootstrapAddresses: nil,
			MDNS:               p2p.MDNSOptions{},
		},
		Timeout: 60 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	a.shardMux = chain.NewShardMux(a.beaconRPC, hn)

	a.hostnode = hn

	genesisTime, err := a.beaconRPC.GetGenesisTime(context.Background(), &empty.Empty{})
	if err != nil {
		return nil, err
	}

	shards, err := utils.ParseRanges(a.Config.TrackShards)
	if err != nil {
		return nil, err
	}

	for _, shard := range shards {
		shardGenesis := primitives.GetGenesisBlockForShard(uint64(shard))

		shardGenesisHash, _ := ssz.HashTreeRoot(shardGenesis)

		_, err := a.shardMux.StartManaging(uint64(shard), chain.ShardChainInitializationParameters{
			RootBlockHash: shardGenesisHash,
			RootSlot:      0,
			GenesisTime:   genesisTime.GenesisTime,
		})
		if err != nil {
			return nil, err
		}
	}

	networkConfig, found := beaconconfig.NetworkIDs[options.NetworkID]
	if !found {
		return nil, fmt.Errorf("could not find network config %s", options.NetworkID)
	}

	go func() {
		err := rpc.Serve(a.Config.RPCProtocol, a.Config.RPCAddress, a.shardMux, &networkConfig, hn)
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

	<-make(chan struct{})

	return nil
}

// Exit exits the module.
func (s *ShardApp) Exit() {}
