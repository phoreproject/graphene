package config

import (
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/utils"
	"google.golang.org/grpc"
)

// Options for the validator module.
type Options struct {
	BeaconRPC  string   `yaml:"beacon_addr" cli:"beacon"`
	ShardRPC   string   `yaml:"shard_addr" cli:"shard"`
	Validators []string `yaml:"validators" cli:"validators"`
	RootKey    string   `yaml:"root_key" cli:"rootkey"`
	RPCListen  string   `yaml:"listen_addr" cli:"listen"`
	NetworkID  string   `yaml:"network_id" cli:"networkid"`
}

// ValidatorConfig is the config passed into the validator app.
type ValidatorConfig struct {
	BeaconConn       *grpc.ClientConn
	ShardConn        *grpc.ClientConn
	NetworkConfig    *config.Config
	ValidatorIndices []uint32
	RootKey          string
}

// ParseValidatorIndices parses validator indices given a user-supplied list of ranges.
func (vc *ValidatorConfig) ParseValidatorIndices(validatorsStrings []string) {
	ranges, err := utils.ParseRanges(validatorsStrings)
	if err != nil {
		panic(err)
	}

	vc.ValidatorIndices = ranges
}
