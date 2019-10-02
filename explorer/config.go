package explorer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"io"

	"github.com/phoreproject/synapse/primitives"

	"github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/p2p"

	beaconapp "github.com/phoreproject/synapse/beacon/module"
)

// Config is the explorer app config.
type Config struct {
	GenesisTime          uint64
	DataDirectory        string
	InitialValidatorList []primitives.InitialValidatorEntry
	NetworkConfig        *config.Config
	Resync               bool
	ListeningAddress     string
	DiscoveryOptions     p2p.DiscoveryOptions
}

// GenerateConfigFromChainConfig generates a new config from the passed in network config
// which should be loaded from a JSON file.
func GenerateConfigFromChainConfig(chainConfig beaconapp.ChainConfig) (*Config, error) {
	c := Config{
		GenesisTime: chainConfig.GenesisTime,
	}

	c.InitialValidatorList = make([]primitives.InitialValidatorEntry, chainConfig.InitialValidators.NumValidators)
	for i := range c.InitialValidatorList {
		validator := chainConfig.InitialValidators.Validators[i]

		pubKeyBytes, err := hex.DecodeString(validator.PubKey)
		if err != nil {
			return nil, err
		}
		var pubKey [96]byte
		copy(pubKey[:], pubKeyBytes)

		sigBytes, err := hex.DecodeString(validator.ProofOfPossession)
		if err != nil {
			return nil, err
		}
		var signature [48]byte
		copy(signature[:], sigBytes)

		withdrawalCredentialsBytes, err := hex.DecodeString(validator.WithdrawalCredentials)
		if err != nil {
			return nil, err
		}
		var withdrawalCredentials [32]byte
		copy(withdrawalCredentials[:], withdrawalCredentialsBytes)

		c.InitialValidatorList[i] = primitives.InitialValidatorEntry{
			PubKey:                pubKey,
			ProofOfPossession:     signature,
			WithdrawalCredentials: withdrawalCredentials,
			WithdrawalShard:       validator.WithdrawalShard,
			DepositSize:           validator.DepositSize,
		}
	}

	c.GenesisTime = chainConfig.GenesisTime

	c.DiscoveryOptions = p2p.NewDiscoveryOptions()

	c.DiscoveryOptions.PeerAddresses = make([]peer.AddrInfo, len(chainConfig.BootstrapPeers))

	networkConfig, found := config.NetworkIDs[chainConfig.NetworkID]
	if !found {
		return nil, fmt.Errorf("error getting network config for ID: %s", chainConfig.NetworkID)
	}
	c.NetworkConfig = &networkConfig

	for i := range c.DiscoveryOptions.PeerAddresses {
		a, err := multiaddr.NewMultiaddr(chainConfig.BootstrapPeers[i])
		if err != nil {
			return nil, err
		}
		peerInfo, err := peer.AddrInfoFromP2pAddr(a)
		if err != nil {
			return nil, err
		}
		c.DiscoveryOptions.PeerAddresses[i] = *peerInfo
	}

	return &c, nil
}

// ReadChainFileToConfig reads a network config from the reader.
func ReadChainFileToConfig(r io.Reader) (*Config, error) {
	var networkConfig beaconapp.ChainConfig

	d := json.NewDecoder(r)
	err := d.Decode(&networkConfig)
	if err != nil {
		return nil, err
	}

	return GenerateConfigFromChainConfig(networkConfig)
}
