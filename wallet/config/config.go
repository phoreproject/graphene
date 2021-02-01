package config

import (
	"google.golang.org/grpc"
)

// Options for the validator module.
type Options struct {
	RPCListen  string `yaml:"rpc_listen_addr" cli:"rpclisten"`
	RelayerRPC string `yaml:"relayer_addr" cli:"relayer"`
}

// WalletConfig is the config passed into the validator app.
type WalletConfig struct {
	RPCAddress  string
	RelayerConn *grpc.ClientConn
}
