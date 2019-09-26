package config

import "google.golang.org/grpc"

// Options are the options passed to the module.
type Options struct {
	RPCListen string `yaml:"listen_addr" cli:"listen"`
	BeaconRPC string `yaml:"beacon_addr" cli:"beacon"`
}

// ShardConfig is the configuration for the shard chain binary.
type ShardConfig struct {
	BeaconConn  *grpc.ClientConn
	RPCProtocol string
	RPCAddress  string
}
