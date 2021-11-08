package primitives

import (
	"errors"

	"github.com/phoreproject/graphene/chainhash"
	"github.com/phoreproject/graphene/pb"
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
	newDp.PublicKey = dp.PubKey[:]
	newDp.ProofOfPossession = dp.ProofOfPossession[:]
	return newDp
}

// DepositParametersFromProto gets the deposit parameters from the protobuf representation.
func DepositParametersFromProto(parameters *pb.DepositParameters) (*DepositParameters, error) {
	if len(parameters.PublicKey) > 96 {
		return nil, errors.New("public key should be 96 bytes")
	}
	if len(parameters.ProofOfPossession) > 48 {
		return nil, errors.New("proof of possession signature should be 48 bytes")
	}
	dp := &DepositParameters{}
	copy(dp.PubKey[:], parameters.PublicKey)
	copy(dp.ProofOfPossession[:], parameters.ProofOfPossession)
	err := dp.WithdrawalCredentials.SetBytes(parameters.WithdrawalCredentials)
	if err != nil {
		return nil, err
	}
	return dp, nil
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

// DepositFromProto gets the deposit from the protobuf representation.
func DepositFromProto(deposit *pb.Deposit) (*Deposit, error) {
	parameters, err := DepositParametersFromProto(deposit.Parameters)
	if err != nil {
		return nil, err
	}
	return &Deposit{*parameters}, nil
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
	newE.Signature = e.Signature[:]
	return newE
}

// ExitFromProto gets the exit from the protobuf representation.
func ExitFromProto(exit *pb.Exit) (*Exit, error) {
	if len(exit.Signature) > 48 {
		return nil, errors.New("exit signature should be 48 bytes")
	}

	e := &Exit{
		Slot:           exit.Slot,
		ValidatorIndex: exit.ValidatorIndex,
	}

	copy(e.Signature[:], exit.Signature)
	return e, nil
}
