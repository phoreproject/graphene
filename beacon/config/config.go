package config

// Config is the config for the blockchain.
type Config struct {
	ShardCount                         int
	TargetCommitteeSize                int
	EjectionBalance                    uint64
	MaxBalanceChurnQuotient            uint64
	BeaconShardNumber                  uint64
	BLSWithdrawalPrefixByte            byte
	MaxCasperVotes                     uint32
	LatestBlockRootsLength             uint64
	LatestRandaoMixesLength            uint64
	InitialForkVersion                 uint64
	InitialSlotNumber                  uint64
	SlotDuration                       uint32
	MinAttestationInclusionDelay       uint64
	EpochLength                        uint64
	CollectivePenaltyCalculationPeriod uint64
	ZeroBalanceValidatorTTL            uint64
	BaseRewardQuotient                 uint64
	WhistleblowerRewardQuotient        uint64
	IncluderRewardQuotient             uint64
	InactivityPenaltyQuotient          uint64
	MaxProposerSlashings               int
	MaxCasperSlashings                 int
	MaxAttestations                    int
	MaxDeposits                        int
	MaxExits                           int
	MaxDeposit                         uint64
	MinDeposit                         uint64
}

// UnitInCoin is the number of base units in 1 coin.
const UnitInCoin = 100000000

// MainNetConfig is the config used on the mainnet
var MainNetConfig = Config{
	ShardCount:                         8,
	TargetCommitteeSize:                3,
	EjectionBalance:                    16,
	MaxBalanceChurnQuotient:            32,
	BeaconShardNumber:                  18446744073709551615,
	BLSWithdrawalPrefixByte:            '\x00',
	MaxCasperVotes:                     1024,
	LatestBlockRootsLength:             8192,
	InitialForkVersion:                 0,
	InitialSlotNumber:                  0,
	SlotDuration:                       6,
	EpochLength:                        8,
	MinAttestationInclusionDelay:       4,
	CollectivePenaltyCalculationPeriod: 1048576,
	ZeroBalanceValidatorTTL:            4194304,
	BaseRewardQuotient:                 1024,
	WhistleblowerRewardQuotient:        512,
	IncluderRewardQuotient:             8,
	InactivityPenaltyQuotient:          17179869184,
	MaxProposerSlashings:               16,
	MaxCasperSlashings:                 16,
	MaxAttestations:                    128,
	MaxDeposits:                        16,
	MaxExits:                           16,
	MaxDeposit:                         64 * UnitInCoin,
	MinDeposit:                         2 * UnitInCoin,
}

// RegtestConfig is the config used for unit tests
var RegtestConfig = Config{
	ShardCount:                         4,
	TargetCommitteeSize:                4,
	EjectionBalance:                    16,
	MaxBalanceChurnQuotient:            32,
	BeaconShardNumber:                  18446744073709551615,
	BLSWithdrawalPrefixByte:            '\x00',
	MaxCasperVotes:                     1024,
	LatestBlockRootsLength:             8192,
	InitialForkVersion:                 0,
	InitialSlotNumber:                  0,
	SlotDuration:                       1,
	EpochLength:                        4,
	MinAttestationInclusionDelay:       0,
	CollectivePenaltyCalculationPeriod: 1048576,
	ZeroBalanceValidatorTTL:            4194304,
	BaseRewardQuotient:                 1024,
	WhistleblowerRewardQuotient:        512,
	IncluderRewardQuotient:             8,
	InactivityPenaltyQuotient:          17179869184,
	MaxProposerSlashings:               16,
	MaxCasperSlashings:                 16,
	MaxAttestations:                    128,
	MaxDeposits:                        16,
	MaxExits:                           16,
	MaxDeposit:                         64 * UnitInCoin,
	MinDeposit:                         2 * UnitInCoin,
}
