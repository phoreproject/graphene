package primitives_test

import (
	"github.com/go-test/deep"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"testing"
)

func TestUpdateWitness_Copy(t *testing.T) {
	baseUpdateWitness := &primitives.UpdateWitness{
		Key:             chainhash.Hash{},
		OldValue:        chainhash.Hash{},
		NewValue:        chainhash.Hash{},
		WitnessBitfield: chainhash.Hash{},
		LastLevel:       0,
		Witnesses:       []chainhash.Hash{
			{},
		},
	}

	copyUpdateWitness := baseUpdateWitness.Copy()

	copyUpdateWitness.Key[0] = 1
	if baseUpdateWitness.Key[0] == 1 {
		t.Fatal("mutating key mutates base")
	}
	copyUpdateWitness.OldValue[0] = 1
	if baseUpdateWitness.OldValue[0] == 1 {
		t.Fatal("mutating OldValue mutates base")
	}
	copyUpdateWitness.NewValue[0] = 1
	if baseUpdateWitness.NewValue[0] == 1 {
		t.Fatal("mutating NewValue mutates base")
	}
	copyUpdateWitness.WitnessBitfield[0] = 1
	if baseUpdateWitness.WitnessBitfield[0] == 1 {
		t.Fatal("mutating WitnessBitfield mutates base")
	}
	copyUpdateWitness.LastLevel = 1
	if baseUpdateWitness.LastLevel == 1 {
		t.Fatal("mutating LastLevel mutates base")
	}
	copyUpdateWitness.Witnesses[0][0] = 1
	if baseUpdateWitness.Witnesses[0][0] == 1 {
		t.Fatal("mutating Witnesses mutates base")
	}
}

func TestUpdateWitness_ToFromProto(t *testing.T) {
	baseUpdateWitness := &primitives.UpdateWitness{
		Key:             chainhash.Hash{1},
		OldValue:        chainhash.Hash{1},
		NewValue:        chainhash.Hash{1},
		WitnessBitfield: chainhash.Hash{1},
		LastLevel:       1,
		Witnesses:       []chainhash.Hash{
			{1},
		},
	}

	updateWitnessProto := baseUpdateWitness.ToProto()
	fromProto, err := primitives.UpdateWitnessFromProto(updateWitnessProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseUpdateWitness); diff != nil {
		t.Fatal(diff)
	}
}


func TestVerificationWitness_Copy(t *testing.T) {
	baseVerificationWitness := &primitives.VerificationWitness{
		Key:             chainhash.Hash{},
		Value:        chainhash.Hash{},
		WitnessBitfield: chainhash.Hash{},
		LastLevel:       0,
		Witnesses:       []chainhash.Hash{
			{},
		},
	}

	copyVerificationWitness := baseVerificationWitness.Copy()

	copyVerificationWitness.Key[0] = 1
	if baseVerificationWitness.Key[0] == 1 {
		t.Fatal("mutating key mutates base")
	}
	copyVerificationWitness.Value[0] = 1
	if baseVerificationWitness.Value[0] == 1 {
		t.Fatal("mutating NewValue mutates base")
	}
	copyVerificationWitness.WitnessBitfield[0] = 1
	if baseVerificationWitness.WitnessBitfield[0] == 1 {
		t.Fatal("mutating WitnessBitfield mutates base")
	}
	copyVerificationWitness.LastLevel = 1
	if baseVerificationWitness.LastLevel == 1 {
		t.Fatal("mutating LastLevel mutates base")
	}
	copyVerificationWitness.Witnesses[0][0] = 1
	if baseVerificationWitness.Witnesses[0][0] == 1 {
		t.Fatal("mutating Witnesses mutates base")
	}
}

func TestVerificationWitness_ToFromProto(t *testing.T) {
	baseVerificationWitness := &primitives.VerificationWitness{
		Key:             chainhash.Hash{1},
		Value:        chainhash.Hash{1},
		WitnessBitfield: chainhash.Hash{1},
		LastLevel:       1,
		Witnesses:       []chainhash.Hash{
			{1},
		},
	}

	verificationWitnessProto := baseVerificationWitness.ToProto()
	fromProto, err := primitives.VerificationWitnessFromProto(verificationWitnessProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseVerificationWitness); diff != nil {
		t.Fatal(diff)
	}
}
