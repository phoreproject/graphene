package app

import "google.golang.org/grpc"

// ShardConfig is the configuration for the shard chain binary.
type ShardConfig struct {
	beaconConn *grpc.ClientConn
}
