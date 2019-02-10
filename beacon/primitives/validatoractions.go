package primitives

import (
	"github.com/phoreproject/synapse/chainhash"
)

// DepositParameters are the parameters the depositer needs
// to provide.
type DepositParameters struct {
	PubKey                [96]byte
	ProofOfPossession     [48]byte
	WithdrawalCredentials chainhash.Hash
}

// Copy returns a copy of the deposit parameters
func (dp *DepositParameters) Copy() DepositParameters {
	return *dp
}

// Deposit is a new deposit from a shard.
type Deposit struct {
	Parameters DepositParameters
}

// Copy returns a copy of the deposit.
func (d Deposit) Copy() Deposit {
	return Deposit{d.Parameters.Copy()}
}

// Exit exits the validator.
type Exit struct {
	Slot           uint64
	ValidatorIndex uint64
	Signature      [48]byte
}

// Copy returns a copy of the exit.
func (e *Exit) Copy() Exit {
	return *e
}
