package primitives

import (
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
)

// InitialValidatorEntry is the validator entry to be added
// at the beginning of a blockchain.
type InitialValidatorEntry struct {
	PubKey                [96]byte
	ProofOfPossession     [48]byte
	WithdrawalShard       uint32
	WithdrawalCredentials chainhash.Hash
	DepositSize           uint64
}

// InitializeState initializes state to the genesis state according to the config.
func InitializeState(c *config.Config, initialValidators []InitialValidatorEntry, genesisTime uint64, skipValidation bool) (*State, error) {
	crosslinks := make([]Crosslink, c.ShardCount)

	for i := 0; i < c.ShardCount; i++ {
		crosslinks[i] = Crosslink{
			Slot:           c.InitialSlotNumber,
			ShardBlockHash: zeroHash,
		}
	}

	recentBlockHashes := make([]chainhash.Hash, c.LatestBlockRootsLength)
	for i := uint64(0); i < c.LatestBlockRootsLength; i++ {
		recentBlockHashes[i] = zeroHash
	}

	initialState := State{
		Slot:        0,
		EpochIndex:  0,
		GenesisTime: genesisTime,
		ForkData: ForkData{
			PreForkVersion:  c.InitialForkVersion,
			PostForkVersion: c.InitialForkVersion,
			ForkSlotNumber:  c.InitialSlotNumber,
		},
		ValidatorRegistry:                  []Validator{},
		ValidatorBalances:                  []uint64{},
		ValidatorRegistryLatestChangeEpoch: 0,
		ValidatorRegistryExitCount:         0,
		ValidatorRegistryDeltaChainTip:     chainhash.Hash{},

		RandaoMix:                 chainhash.Hash{},
		ShardAndCommitteeForSlots: [][]ShardAndCommittee{},

		PreviousJustifiedEpoch: 0,
		JustifiedEpoch:         0,
		JustificationBitfield:  0,
		FinalizedEpoch:         0,

		LatestCrosslinks:          crosslinks,
		PreviousCrosslinks:        crosslinks,
		LatestBlockHashes:         recentBlockHashes,
		CurrentEpochAttestations:  []PendingAttestation{},
		PreviousEpochAttestations: []PendingAttestation{},
		BatchedBlockRoots:         []chainhash.Hash{},
	}

	for _, deposit := range initialValidators {
		pub, err := bls.DeserializePublicKey(deposit.PubKey)
		if err != nil {
			return nil, err
		}
		validatorIndex, err := initialState.ProcessDeposit(pub, deposit.DepositSize, deposit.ProofOfPossession, deposit.WithdrawalCredentials, skipValidation, c)
		if err != nil {
			return nil, err
		}
		if initialState.GetEffectiveBalance(validatorIndex, c) == c.MaxDeposit {
			err := initialState.UpdateValidatorStatus(validatorIndex, Active, c)
			if err != nil {
				return nil, err
			}
		}
	}

	initialShuffling := GetNewShuffling(zeroHash, initialState.ValidatorRegistry, 0, c)
	initialState.ShardAndCommitteeForSlots = append(initialShuffling, initialShuffling...)

	return &initialState, nil
}
