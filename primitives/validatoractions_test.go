package primitives_test

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/phoreproject/graphene/chainhash"
	"github.com/phoreproject/graphene/primitives"
)

func TestDepositParameters_Copy(t *testing.T) {
	baseForkData := &primitives.DepositParameters{
		PubKey:                [96]byte{},
		ProofOfPossession:     [48]byte{},
		WithdrawalCredentials: chainhash.Hash{},
	}

	copyForkData := baseForkData.Copy()

	copyForkData.PubKey[0] = 1
	if baseForkData.PubKey[0] == 1 {
		t.Fatal("mutating pubkey mutates base")
	}

	copyForkData.ProofOfPossession[0] = 1
	if baseForkData.ProofOfPossession[0] == 1 {
		t.Fatal("mutating proofOfPossession mutates base")
	}

	copyForkData.WithdrawalCredentials[0] = 1
	if baseForkData.WithdrawalCredentials[0] == 1 {
		t.Fatal("mutating withdrawalCredentials mutates base")
	}
}

func TestDepositParameters_ToFromProto(t *testing.T) {
	baseDepositParameters := &primitives.DepositParameters{
		PubKey:                [96]byte{1},
		ProofOfPossession:     [48]byte{1},
		WithdrawalCredentials: chainhash.Hash{1},
	}

	depositParametersProto := baseDepositParameters.ToProto()
	fromProto, err := primitives.DepositParametersFromProto(depositParametersProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseDepositParameters); diff != nil {
		t.Fatal(diff)
	}
}

func TestDeposit_Copy(t *testing.T) {
	baseDeposit := &primitives.Deposit{Parameters: primitives.DepositParameters{}}

	copyDeposit := baseDeposit.Copy()

	copyDeposit.Parameters.WithdrawalCredentials[0] = 1
	if baseDeposit.Parameters.WithdrawalCredentials[0] == 1 {
		t.Fatal("mutating depositParameters mutates base")
	}
}

func TestDeposit_ToFromProto(t *testing.T) {
	baseDeposit := &primitives.Deposit{
		Parameters: primitives.DepositParameters{
			PubKey: [96]byte{},
		},
	}

	depositProto := baseDeposit.ToProto()
	fromProto, err := primitives.DepositFromProto(depositProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseDeposit); diff != nil {
		t.Fatal(diff)
	}
}

func TestExit_Copy(t *testing.T) {
	baseExit := &primitives.Exit{
		Slot:           0,
		ValidatorIndex: 0,
		Signature:      [48]byte{},
	}

	copyExit := baseExit.Copy()

	copyExit.Slot = 1
	if baseExit.Slot == 1 {
		t.Fatal("mutating slot mutates base")
	}

	copyExit.ValidatorIndex = 1
	if baseExit.ValidatorIndex == 1 {
		t.Fatal("mutating slot mutates base")
	}

	copyExit.Signature[0] = 1
	if baseExit.Signature[0] == 1 {
		t.Fatal("mutating signature mutates base")
	}
}

func TestExit_ToFromProto(t *testing.T) {
	baseExit := &primitives.Exit{
		Slot:           1,
		ValidatorIndex: 1,
		Signature:      [48]byte{1},
	}

	exitProto := baseExit.ToProto()
	fromProto, err := primitives.ExitFromProto(exitProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseExit); diff != nil {
		t.Fatal(diff)
	}
}
