package config

import (
	"github.com/phoreproject/synapse/p2p"
	"google.golang.org/grpc"
)

// Options are the options passed to the module.
type Options struct {
	RPCListen          string   `yaml:"listen_addr" cli:"rpclisten"`
	BeaconRPC          string   `yaml:"beacon_addr" cli:"beacon"`
	TrackShards        []string `yaml:"track_shards" cli:"track"`
	NetworkID          string   `yaml:"network_id" cli:"networkid"`
	P2PListen          string   `yaml:"p2p_listen_addr" cli:"listen"`
	InitialConnections []string `yaml:"initial_connections" cli:"connect"`
}

// ShardConfig is the configuration for the shard chain binary.
type ShardConfig struct {
	BeaconConn  *grpc.ClientConn
	RPCProtocol string
	RPCAddress  string
	TrackShards []string
	P2PListen   string

	DiscoveryOptions p2p.ConnectionManagerOptions
}
