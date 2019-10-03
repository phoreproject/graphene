package module

import (
	manet "github.com/multiformats/go-multiaddr-net"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/relayer/config"
)

// RelayerModule is a module that handles processing and packaging transactions into packages.
type RelayerModule struct {
	Options config.Options
	RPCServer pb.RelayerRPCServer
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