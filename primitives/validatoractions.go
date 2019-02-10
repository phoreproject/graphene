package primitives

import (
	"github.com/phoreproject/synapse/chainhash"
	pb "github.com/phoreproject/synapse/pb"
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

// ToProto gets the protobuf representation of the deposit parameters.
func (dp *DepositParameters) ToProto() *pb.DepositParameters {
	newDp := &pb.DepositParameters{
		WithdrawalCredentials: dp.WithdrawalCredentials[:],
	}
	copy(newDp.PublicKey, dp.PubKey[:])
	copy(newDp.ProofOfPossession, dp.ProofOfPossession[:])
	return newDp
}

// Deposit is a new deposit from a shard.
type Deposit struct {
	Parameters DepositParameters
}

// ToProto gets the protobuf representation of the deposit.
func (d Deposit) ToProto() *pb.Deposit {
	return &pb.Deposit{
		Parameters: d.Parameters.ToProto(),
	}
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

// ToProto gets the protobuf representation of the exit.
func (e *Exit) ToProto() *pb.Exit {
	newE := &pb.Exit{
		Slot:           e.Slot,
		ValidatorIndex: e.ValidatorIndex,
	}
	copy(newE.Signature, e.Signature[:])
	return newE
}
