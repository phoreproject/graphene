package module

import (
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/beacon/chainfile"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/p2p"
)

// GenerateConfigFromChainConfig generates a new config from the passed in network config
// which should be loaded from a JSON file.
func GenerateConfigFromChainConfig(chainConfig *chainfile.ChainConfig) (*Config, error) {
	c := NewConfig()

	initialValidators, err := chainConfig.GetInitialValidators()
	if err != nil {
		return nil, err
	}
	c.InitialValidatorList = initialValidators

	c.GenesisTime = chainConfig.GenesisTime

	c.DiscoveryOptions = p2p.NewConnectionManagerOptions()

	c.DiscoveryOptions.BootstrapAddresses = make([]peer.AddrInfo, len(chainConfig.BootstrapPeers))

	networkConfig, found := config.NetworkIDs[chainConfig.NetworkID]
	if !found {
		return nil, fmt.Errorf("error getting network config for ID: %s", chainConfig.NetworkID)
	}
	c.NetworkConfig = &networkConfig

	for i := range c.DiscoveryOptions.BootstrapAddresses {
		a, err := multiaddr.NewMultiaddr(chainConfig.BootstrapPeers[i])
		if err != nil {
			return nil, err
		}
		peerInfo, err := peer.AddrInfoFromP2pAddr(a)
		if err != nil {
			return nil, err
		}
		c.DiscoveryOptions.BootstrapAddresses[i] = *peerInfo
	}

	return &c, nil
}

// ReadChainFileToConfig reads a network config from the reader.
func ReadChainFileToConfig(r io.Reader) (*Config, error) {
	networkConfig, err := chainfile.ReadChainFile(r)
	if err != nil {
		return nil, err
	}

	return GenerateConfigFromChainConfig(networkConfig)
}
