package primitives_test

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/phoreproject/graphene/chainhash"
	"github.com/phoreproject/graphene/primitives"
)

func TestValidator_Copy(t *testing.T) {
	baseValidator := &primitives.Validator{
		Pubkey:                  [96]byte{},
		WithdrawalCredentials:   chainhash.Hash{},
		Status:                  0,
		LatestStatusChangeSlot:  0,
		ExitCount:               0,
		LastPoCChangeSlot:       0,
		SecondLastPoCChangeSlot: 0,
	}

	copyValidator := baseValidator.Copy()

	copyValidator.Pubkey[0] = 1
	if baseValidator.Pubkey[0] == 1 {
		t.Fatal("mutating pubkey mutates base")
	}

	copyValidator.WithdrawalCredentials[0] = 1
	if baseValidator.WithdrawalCredentials[0] == 1 {
		t.Fatal("mutating withdrawalCredentials mutates base")
	}

	copyValidator.Status = 1
	if baseValidator.Status == 1 {
		t.Fatal("mutating status mutates base")
	}

	copyValidator.LatestStatusChangeSlot = 1
	if baseValidator.LatestStatusChangeSlot == 1 {
		t.Fatal("mutating LatestStatusChangeSlot mutates base")
	}

	copyValidator.ExitCount = 1
	if baseValidator.ExitCount == 1 {
		t.Fatal("mutating ExitCount mutates base")
	}

	copyValidator.LastPoCChangeSlot = 1
	if baseValidator.LastPoCChangeSlot == 1 {
		t.Fatal("mutating LastPoCChangeSlot mutates base")
	}

	copyValidator.SecondLastPoCChangeSlot = 1
	if baseValidator.SecondLastPoCChangeSlot == 1 {
		t.Fatal("mutating SecondLastPoCChangeSlot mutates base")
	}
}

func TestValidator_ToFromProto(t *testing.T) {
	baseValidator := &primitives.Validator{
		Pubkey:                  [96]byte{1},
		WithdrawalCredentials:   chainhash.Hash{},
		Status:                  0,
		LatestStatusChangeSlot:  0,
		ExitCount:               0,
		LastPoCChangeSlot:       0,
		SecondLastPoCChangeSlot: 0,
	}

	validatorProto := baseValidator.ToProto()
	fromProto, err := primitives.ValidatorFromProto(validatorProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseValidator); diff != nil {
		t.Fatal(diff)
	}
}

func TestValidatorProof_Copy(t *testing.T) {
	baseProof := &primitives.ValidatorProof{
		ShardID:        0,
		ValidatorIndex: 0,
		PublicKey:      [96]byte{},
		Proof: primitives.VerificationWitness{
			Key:             chainhash.Hash{},
			Value:           chainhash.Hash{},
			WitnessBitfield: chainhash.Hash{},
			Witnesses:       nil,
			LastLevel:       0,
		},
	}

	copyProof := baseProof.Copy()

	copyProof.ShardID = 1
	if baseProof.ShardID == 1 {
		t.Fatal("mutating ShardID mutates base")
	}

	copyProof.ValidatorIndex = 1
	if baseProof.ValidatorIndex == 1 {
		t.Fatal("mutating ValidatorIndex mutates base")
	}

	copyProof.PublicKey[0] = 1
	if baseProof.PublicKey[0] == 1 {
		t.Fatal("mutating PublicKey mutates base")
	}

	copyProof.Proof.LastLevel = 1
	if baseProof.Proof.LastLevel == 1 {
		t.Fatal("mutating Proof mutates base")
	}
}

func TestValidatorProof_ToFromProto(t *testing.T) {
	baseProof := &primitives.ValidatorProof{
		ShardID:        1,
		ValidatorIndex: 2,
		PublicKey:      [96]byte{3},
		Proof: primitives.VerificationWitness{
			Key:             chainhash.Hash{4},
			Value:           chainhash.Hash{},
			WitnessBitfield: chainhash.Hash{},
			Witnesses:       []chainhash.Hash{},
			LastLevel:       0,
		},
	}

	proofProto := baseProof.ToProto()
	fromProto, err := primitives.ValidatorProofFromProto(proofProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseProof); diff != nil {
		t.Fatal(diff)
	}
}
