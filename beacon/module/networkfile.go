package module

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/graphene/beacon/config"
	"github.com/phoreproject/graphene/p2p"
	"github.com/phoreproject/graphene/primitives"
)

// InitialValidatorInformation is the encoded information from the JSON file of an
// initial validator.
type InitialValidatorInformation struct {
	PubKey                string
	ProofOfPossession     string
	WithdrawalShard       uint32
	WithdrawalCredentials string
	DepositSize           uint64
	ID                    uint32
}

// InitialValidatorList is a list of initial validators and the number of validators
type InitialValidatorList struct {
	NumValidators int
	Validators    []InitialValidatorInformation
}

// ChainConfig is the JSON encoded information about the network.
type ChainConfig struct {
	GenesisTime       uint64
	BootstrapPeers    []string
	NetworkID         string
	InitialValidators InitialValidatorList
}

// GenerateConfigFromChainConfig generates a new config from the passed in network config
// which should be loaded from a JSON file.
func GenerateConfigFromChainConfig(chainConfig ChainConfig) (*Config, error) {
	c := NewConfig()

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
	var networkConfig ChainConfig

	d := json.NewDecoder(r)
	err := d.Decode(&networkConfig)
	if err != nil {
		return nil, err
	}

	return GenerateConfigFromChainConfig(networkConfig)
}
