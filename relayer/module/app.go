package module

import (
	"context"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/relayer/mempool"
	shardp2p "github.com/phoreproject/synapse/relayer/p2p"
	"github.com/phoreproject/synapse/relayer/rpc"
	"github.com/phoreproject/synapse/relayer/shardrelayer"
	"github.com/phoreproject/synapse/shard/execution"
	"github.com/phoreproject/synapse/shard/transfer"
	"github.com/phoreproject/synapse/utils"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/relayer/config"
)

// RelayerModule is a module that handles processing and packaging transactions into packages.
type RelayerModule struct {
	Options   config.Options
	RPCServer pb.RelayerRPCServer
	hostnode  *p2p.HostNode
	relayers  []*shardrelayer.ShardRelayer
}

// NewRelayerModule creates a new relayer module.
func NewRelayerModule(o config.Options) (*RelayerModule, error) {
	r := &RelayerModule{
		Options: o,
	}

	if err := r.createRPCServer(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *RelayerModule) createRPCServer() error {
	rpcListen, err := ma.NewMultiaddr(r.Options.RPCListen)
	if err != nil {
		return err
	}

	rpcListenAddr, err := manet.ToNetAddr(rpcListen)
	if err != nil {
		return err
	}

	mempools := make(map[uint64]*mempool.ShardMempool)
	for _, r := range r.relayers {
		mempools[r.GetShardID()] = r.GetMempool()
	}

	go func() {
		err := rpc.Serve(rpcListenAddr.Network(), rpcListenAddr.String(), mempools)
		if err != nil {
			logrus.Error(err)
		}
	}()
	return nil
}

// Run runs the relayer module.
func (r *RelayerModule) Run() error {
	shardAddr, err := utils.MultiaddrStringToDialString(r.Options.ShardRPC)
	if err != nil {
		return err
	}

	shards, err := utils.ParseRanges(r.Options.Shards)
	if err != nil {
		return err
	}

	shardConn, err := grpc.Dial(shardAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	addr, err := ma.NewMultiaddr(r.Options.P2PListen)
	if err != nil {
		return err
	}

	hn, err := p2p.NewHostNode(context.Background(), p2p.HostNodeOptions{
		ListenAddresses: []ma.Multiaddr{
			addr,
		},
		PrivateKey:         nil,
		ConnManagerOptions: p2p.ConnectionManagerOptions{},
		Timeout:            0,
	})
	if err != nil {
		return err
	}

	shardRPC := pb.NewShardRPCClient(shardConn)

	// TODO: save state to disk
	r.relayers = make([]*shardrelayer.ShardRelayer, len(shards))
	for i, s := range shards {
		stateDB := csmt.NewInMemoryTreeDB()
		genesisBlock := primitives.GetGenesisBlockForShard(uint64(s))
		genesisHash, _ := ssz.HashTreeRoot(genesisBlock)

		m := mempool.NewShardMempool(stateDB, 0, genesisHash, execution.ShardInfo{
			CurrentCode: transfer.Code,
			ShardID:     uint32(s),
		})

		relayerP2P, err := shardp2p.NewRelayerSyncManager(hn, m, uint64(s))
		if err != nil {
			return err
		}

		r.relayers[i] = shardrelayer.NewShardRelayer(uint64(s), shardRPC, m, relayerP2P)
	}

	for _, relayer := range r.relayers {
		relayer.ListenForActions()
	}

	<-make(chan struct{})

	return nil
}
