package config

import (
	"strconv"
	"strings"

	"github.com/phoreproject/synapse/beacon/config"
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
	var validatorIndices []uint32
	validatorIndicesMap := map[int]struct{}{}
	for _, s := range validatorsStrings {
		if !strings.ContainsRune(s, '-') {
			i, err := strconv.Atoi(s)
			if err != nil {
				panic("invalid validators parameter")
			}
			validatorIndicesMap[i] = struct{}{}
			validatorIndices = append(validatorIndices, uint32(i))
		} else {
			parts := strings.SplitN(s, "-", 2)
			if len(parts) != 2 {
				panic("invalid validators parameter")
			}
			first, err := strconv.Atoi(parts[0])
			if err != nil {
				panic("invalid validators parameter")
			}
			second, err := strconv.Atoi(parts[1])
			if err != nil {
				panic("invalid validators parameter")
			}
			for i := first; i <= second; i++ {
				validatorIndices = append(validatorIndices, uint32(i))
				validatorIndicesMap[i] = struct{}{}
			}
		}
	}

	vc.ValidatorIndices = validatorIndices
}
