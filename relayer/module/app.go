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
	"github.com/phoreproject/synapse/relayer/shardrelayer"
	"github.com/phoreproject/synapse/shard/execution"
	"github.com/phoreproject/synapse/shard/transfer"
	"github.com/phoreproject/synapse/utils"
	"github.com/prysmaticlabs/go-ssz"
	"google.golang.org/grpc"

	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/relayer/config"
)

// RelayerModule is a module that handles processing and packaging transactions into packages.
type RelayerModule struct {
	Options config.Options
	RPCServer pb.RelayerRPCServer
	hostnode *p2p.HostNode
	relayers []*shardrelayer.ShardRelayer
}

// NewRelayerModule creates a new relayer module.
func NewRelayerModule(o config.Options) (*RelayerModule, error) {
	shards, err := utils.ParseRanges(o.Shards)
	if err != nil {
		return nil, err
	}

	shardConn, err := grpc.Dial(o.ShardRPC)
	if err != nil {
		return nil, err
	}

	addr, err := ma.NewMultiaddr(o.P2PListen)
	if err != nil {
		return nil, err
	}

	hn, err := p2p.NewHostNode(context.Background(), p2p.HostNodeOptions{
		ListenAddresses:    []ma.Multiaddr{
			addr,
		},
		PrivateKey:         nil,
		ConnManagerOptions: p2p.ConnectionManagerOptions{},
		Timeout:            0,
	})
	if err != nil {
		return nil, err
	}

	shardRPC := pb.NewShardRPCClient(shardConn)

	// TODO: save state to disk
	relayers := make([]*shardrelayer.ShardRelayer, len(shards))
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
			return nil, err
		}

		relayers[i] = shardrelayer.NewShardRelayer(uint64(s), shardRPC, m, relayerP2P)
	}

	r := &RelayerModule{
		Options: o,
		relayers: relayers,
		hostnode: hn,
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

	_, err = manet.ToNetAddr(rpcListen)
	if err != nil {
		return err
	}

	//go func() {
	//	err := rpc.Serve(rpcListenAddr.Network(), rpcListenAddr.String(), app.blockchain, app.hostNode, app.mempool)
	//	if err != nil {
	//		panic(err)
	//	}
	//}()

	return nil
}

// Run runs the relayer module.
func (r *RelayerModule) Run() error {

	return nil
}