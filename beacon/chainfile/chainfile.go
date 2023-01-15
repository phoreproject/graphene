package chainfile

import (
	"encoding/hex"
	"encoding/json"
	"io"

	"github.com/phoreproject/synapse/primitives"
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

// ReadChainFile reads a network config from the reader.
func ReadChainFile(r io.Reader) (*ChainConfig, error) {
	var networkConfig ChainConfig

	d := json.NewDecoder(r)
	err := d.Decode(&networkConfig)
	if err != nil {
		return nil, err
	}

	return &networkConfig, nil
}

func (chainConfig *ChainConfig) GetInitialValidators() ([]primitives.InitialValidatorEntry, error) {
	initialValidators := make([]primitives.InitialValidatorEntry, chainConfig.InitialValidators.NumValidators)
	for i := range initialValidators {
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

		initialValidators[i] = primitives.InitialValidatorEntry{
			PubKey:                pubKey,
			ProofOfPossession:     signature,
			WithdrawalCredentials: withdrawalCredentials,
			WithdrawalShard:       validator.WithdrawalShard,
			DepositSize:           validator.DepositSize,
		}
	}

	return initialValidators, nil
}
