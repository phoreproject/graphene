package module

import (
	"context"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multiaddr-net"
	"github.com/phoreproject/synapse/wallet/config"
	"github.com/phoreproject/synapse/wallet/rpc"
	"google.golang.org/grpc"
)

// ValidatorApp is the app to run the validator runtime.
type WalletApp struct {
	config config.WalletConfig
	ctx    context.Context
	cancel context.CancelFunc
}

// NewWalletApp creates a new validator app from the config.
func NewWalletApp(options config.Options) (*WalletApp, error) {
	// TODO Really important!! Implement secure communication layer!
	relayerConn, err := grpc.Dial(options.RelayerRPC, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	c := config.WalletConfig{
		RelayerConn: relayerConn,
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &WalletApp{
		config: c,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (w *WalletApp) createRPCServer() error {
	rpcListen, err := ma.NewMultiaddr(w.config.RPCAddress)
	if err != nil {
		return err
	}

	rpcListenAddr, err := manet.ToNetAddr(rpcListen)
	if err != nil {
		return err
	}

	walletRpc := rpc.NewWalletRPC(w.config.RelayerConn)

	go func() {
		err := rpc.Serve(rpcListenAddr.Network(), rpcListenAddr.String(), walletRpc)
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

// Run starts the validator app.
func (w *WalletApp) Run() error {
	return nil
}

// Exit exits the validator app.
func (w *WalletApp) Exit() {
	w.cancel()
}
