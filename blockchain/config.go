package blockchain

// Config is the config for the blockchain.
type Config struct {
	CycleLength                       int
	DepositSize                       uint64
	MinCommitteeSize                  int
	ShardCount                        int
	RandaoSlotsPerLayer               int
	BaseRewardQuotient                uint64
	SqrtEDropTime                     uint64
	WithdrawalPeriod                  uint64
	MinimumDepositSize                uint64
	MinimumValidatorSetChangeInterval uint64
	MaxValidatorChurnQuotient         uint64
}

// UnitInCoin is the number of base units in 1 coin.
const UnitInCoin = 100000000

// MainNetConfig is the config used on the mainnet
var MainNetConfig = Config{
	CycleLength:                       64,
	DepositSize:                       100 * UnitInCoin,
	MinCommitteeSize:                  128,
	ShardCount:                        5,
	RandaoSlotsPerLayer:               4192,
	BaseRewardQuotient:                85000,
	SqrtEDropTime:                     65536,
	WithdrawalPeriod:                  524288,
	MinimumDepositSize:                50 * UnitInCoin,
	MinimumValidatorSetChangeInterval: 256,
	MaxValidatorChurnQuotient:         32,
}

const (
	// PendingActivation is when the validator is queued to be
	// added to the validator set.
	PendingActivation = iota

	// Active is when the validator is part of the validator
	// set.
	Active

	// PendingExit is when a validator is queued to exit the
	// validator set.
	PendingExit

	// PendingWithdraw is when a validator withdraws from the beacon
	// chain.
	PendingWithdraw

	// Withdrawn is the state after withdrawing from the validator.
	Withdrawn

	// Penalized is when the validator is penalized for being dishonest.
	Penalized = 127
)
