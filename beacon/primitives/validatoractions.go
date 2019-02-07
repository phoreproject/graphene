package primitives

import (
	"io"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

// DepositParameters are the parameters the depositer needs
// to provide.
type DepositParameters struct {
	PubKey                bls.PublicKey
	ProofOfPossession     bls.Signature
	WithdrawalCredentials chainhash.Hash
	RandaoCommitment      chainhash.Hash
}

// EncodeSSZ implements Encodable
func (dp DepositParameters) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, dp.PubKey); err != nil {
		return err
	}
	if err := ssz.Encode(writer, dp.ProofOfPossession); err != nil {
		return err
	}
	if err := ssz.Encode(writer, dp.WithdrawalCredentials); err != nil {
		return err
	}
	if err := ssz.Encode(writer, dp.RandaoCommitment); err != nil {
		return err
	}
	return nil
}

// EncodeSSZSize implements Encodable
func (dp DepositParameters) EncodeSSZSize() (uint32, error) {
	var sizeOfpubKey, sizeOfproofOfPossession, sizeOfwithdrawalCredentials, sizeOfrandaoCommitment uint32
	var err error
	if sizeOfpubKey, err = ssz.EncodeSize(dp.PubKey); err != nil {
		return 0, err
	}
	if sizeOfproofOfPossession, err = ssz.EncodeSize(dp.ProofOfPossession); err != nil {
		return 0, err
	}
	if sizeOfwithdrawalCredentials, err = ssz.EncodeSize(dp.WithdrawalCredentials); err != nil {
		return 0, err
	}
	if sizeOfrandaoCommitment, err = ssz.EncodeSize(dp.RandaoCommitment); err != nil {
		return 0, err
	}
	return sizeOfpubKey + sizeOfproofOfPossession + sizeOfwithdrawalCredentials + sizeOfrandaoCommitment, nil
}

// DecodeSSZ implements Decodable
func (dp DepositParameters) DecodeSSZ(reader io.Reader) error {
	dp.PubKey = bls.PublicKey{}
	if err := ssz.Decode(reader, &dp.PubKey); err != nil {
		return err
	}
	dp.ProofOfPossession = bls.Signature{}
	if err := ssz.Decode(reader, &dp.ProofOfPossession); err != nil {
		return err
	}
	dp.WithdrawalCredentials = chainhash.Hash{}
	if err := ssz.Decode(reader, &dp.WithdrawalCredentials); err != nil {
		return err
	}
	dp.RandaoCommitment = chainhash.Hash{}
	if err := ssz.Decode(reader, &dp.RandaoCommitment); err != nil {
		return err
	}
	return nil
}

// Copy returns a copy of the deposit parameters
func (dp *DepositParameters) Copy() DepositParameters {
	newDP := *dp
	newDP.PubKey = dp.PubKey.Copy()
	newSig := dp.ProofOfPossession.Copy()
	newDP.ProofOfPossession = *newSig
	return newDP
}

// Deposit is a new deposit from a shard.
type Deposit struct {
	Parameters DepositParameters
}

// EncodeSSZ implements Encodable
func (d Deposit) EncodeSSZ(writer io.Writer) error {
	if err := ssz.Encode(writer, d.Parameters); err != nil {
		return err
	}
	return nil
}

// EncodeSSZSize implements Encodable
func (d Deposit) EncodeSSZSize() (uint32, error) {
	var sizeOfparameters uint32
	var err error
	if sizeOfparameters, err = ssz.EncodeSize(d.Parameters); err != nil {
		return 0, err
	}
	return sizeOfparameters, nil
}

// DecodeSSZ implements Decodable
func (d Deposit) DecodeSSZ(reader io.Reader) error {
	d.Parameters = DepositParameters{}
	if err := ssz.Decode(reader, &d.Parameters); err != nil {
		return err
	}
	return nil
}

// Copy returns a copy of the deposit.
func (d Deposit) Copy() Deposit {
	return Deposit{d.Parameters.Copy()}
}

// Exit exits the validator.
type Exit struct {
	Slot           uint64
	ValidatorIndex uint64
	Signature      bls.Signature
}

// Copy returns a copy of the exit.
func (e *Exit) Copy() Exit {
	newSig := e.Signature.Copy()
	return Exit{
		e.Slot,
		e.ValidatorIndex,
		*newSig,
	}
}
