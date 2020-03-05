package primitives_test

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/chainhash"
)

func TestAttestationData_Equals(t *testing.T) {
	baseAtt := &primitives.AttestationData{
		Slot:                0,
		Shard:               0,
		BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
		SourceEpoch:         0,
		SourceHash:          chainhash.HashH([]byte("epoch")),
		ShardBlockHash:      chainhash.HashH([]byte("shard")),
		ShardStateHash: chainhash.HashH([]byte("shardstate")),
		LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
		TargetEpoch:         0,
		TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
	}

	testCases := []struct {
		attData    *primitives.AttestationData
		equalsBase bool
	}{
		{
			attData: &primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				ShardStateHash: chainhash.HashH([]byte("shardstate")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         0,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			equalsBase: true,
		},
		{
			attData: &primitives.AttestationData{
				Slot:                1,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         0,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			equalsBase: false,
		},
		{
			attData: &primitives.AttestationData{
				Slot:                0,
				Shard:               1,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         0,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			equalsBase: false,
		},
		{
			attData: &primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon1")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         0,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			equalsBase: false,
		},
		{
			attData: &primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         1,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         0,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			equalsBase: false,
		},
		{
			attData: &primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch1")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         0,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			equalsBase: false,
		},
		{
			attData: &primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard1")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         0,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			equalsBase: false,
		},
		{
			attData: &primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink1")),
				TargetEpoch:         0,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			equalsBase: false,
		},
		{
			attData: &primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         1,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			equalsBase: false,
		},
		{
			attData: &primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         0,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash1")),
			},
			equalsBase: false,
		},
		{
			attData: &primitives.AttestationData{
				Slot:                0,
				Shard:               0,
				BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
				SourceEpoch:         0,
				SourceHash:          chainhash.HashH([]byte("epoch")),
				ShardBlockHash:      chainhash.HashH([]byte("shard")),
				ShardStateHash:      chainhash.HashH([]byte("shardstate1")),
				LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
				TargetEpoch:         0,
				TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
			},
			equalsBase: false,
		},
	}

	for _, c := range testCases {
		if baseAtt.Equals(c.attData) != c.equalsBase {
			t.Fatalf("AttestationData.Equals did not validate for test case: %#v", c.attData)
		}
	}
}

func TestAttestationData_Copy(t *testing.T) {
	baseAtt := &primitives.AttestationData{
		Slot:                0,
		Shard:               0,
		BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
		SourceEpoch:         0,
		SourceHash:          chainhash.HashH([]byte("epoch")),
		ShardBlockHash:      chainhash.HashH([]byte("shard")),
		ShardStateHash:      chainhash.HashH([]byte("shardstate")),
		LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
		TargetEpoch:         0,
		TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
	}

	copyAtt := baseAtt.Copy()
	baseAtt.ShardBlockHash = chainhash.HashH([]byte("beacon2"))

	if copyAtt.Equals(baseAtt) {
		t.Fatal("mutating attestation mutated a copy")
	}
}

func TestAttestationDataToFromProto(t *testing.T) {
	baseAtt := &primitives.AttestationData{
		Slot:                0,
		Shard:               0,
		BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
		SourceEpoch:         0,
		SourceHash:          chainhash.HashH([]byte("epoch")),
		ShardBlockHash:      chainhash.HashH([]byte("shard")),
		ShardStateHash:      chainhash.HashH([]byte("shardstate")),
		LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
		TargetEpoch:         0,
		TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
	}

	baseAttProto := baseAtt.ToProto()
	fromProto, err := primitives.AttestationDataFromProto(baseAttProto)
	if err != nil {
		t.Fatal(err)
	}
	if !fromProto.Equals(baseAtt) {
		t.Fatal("AttestationData did not match after proto/unproto")
	}
}

func TestAttestation_Copy(t *testing.T) {
	baseAtt := &primitives.Attestation{
		Data: primitives.AttestationData{
			Slot:                0,
			Shard:               0,
			BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
			SourceEpoch:         0,
			SourceHash:          chainhash.HashH([]byte("epoch")),
			ShardBlockHash:      chainhash.HashH([]byte("shard")),
			ShardStateHash:      chainhash.HashH([]byte("shardstate")),
			LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
			TargetEpoch:         1,
			TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
		},
		ParticipationBitfield: []uint8{0, 0, 0},
		CustodyBitfield:       []uint8{0, 0, 0},
		AggregateSig:          [48]byte{},
	}

	copyAtt := baseAtt.Copy()
	copyAtt.ParticipationBitfield[0] = 1

	if baseAtt.ParticipationBitfield[0] == 1 {
		t.Fatal("mutating copy participationBitfield mutated base")
	}

	copyAtt.CustodyBitfield[0] = 1

	if baseAtt.CustodyBitfield[0] == 1 {
		t.Fatal("mutating copy custodyBitfield mutated base")
	}

	copyAtt.AggregateSig[0] = 1

	if baseAtt.AggregateSig[0] == 1 {
		t.Fatal("mutating copy aggregateSig mutated base")
	}

	copyAtt.Data.Shard = 1

	if baseAtt.Data.Shard == 1 {
		t.Fatal("mutating copy data mutated base")
	}
}

func TestAttestationToFromProto(t *testing.T) {
	baseAtt := &primitives.Attestation{
		Data: primitives.AttestationData{
			Slot:                0,
			Shard:               0,
			BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
			SourceEpoch:         0,
			SourceHash:          chainhash.HashH([]byte("epoch")),
			ShardBlockHash:      chainhash.HashH([]byte("shard")),
			LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
			TargetEpoch:         1,
			TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
		},
		ParticipationBitfield: []uint8{0, 1, 0},
		CustodyBitfield:       []uint8{0, 1, 0},
		AggregateSig:          [48]byte{0, 1},
	}

	baseAttProto := baseAtt.ToProto()
	fromProto, err := primitives.AttestationFromProto(baseAttProto)
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(fromProto, baseAtt); diff != nil {
		t.Fatal(diff)
	}
}

func TestPendingAttestation_Copy(t *testing.T) {
	baseAtt := &primitives.PendingAttestation{
		Data: primitives.AttestationData{
			Slot:                0,
			Shard:               0,
			BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
			SourceEpoch:         0,
			SourceHash:          chainhash.HashH([]byte("epoch")),
			ShardBlockHash:      chainhash.HashH([]byte("shard")),
			LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
			TargetEpoch:         1,
			TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
		},
		ParticipationBitfield: []uint8{0, 0, 0},
		CustodyBitfield:       []uint8{0, 0, 0},
	}

	copyAtt := baseAtt.Copy()
	copyAtt.ParticipationBitfield[0] = 1

	if baseAtt.ParticipationBitfield[0] == 1 {
		t.Fatal("mutating copy participationBitfield mutated base")
	}

	copyAtt.CustodyBitfield[0] = 1

	if baseAtt.CustodyBitfield[0] == 1 {
		t.Fatal("mutating copy custodyBitfield mutated base")
	}

	copyAtt.Data.Shard = 1

	if baseAtt.Data.Shard == 1 {
		t.Fatal("mutating copy data mutated base")
	}
}

func TestPendingAttestationToFromProto(t *testing.T) {
	baseAtt := &primitives.PendingAttestation{
		Data: primitives.AttestationData{
			Slot:                0,
			Shard:               0,
			BeaconBlockHash:     chainhash.HashH([]byte("beacon")),
			SourceEpoch:         0,
			SourceHash:          chainhash.HashH([]byte("epoch")),
			ShardBlockHash:      chainhash.HashH([]byte("shard")),
			LatestCrosslinkHash: chainhash.HashH([]byte("crosslink")),
			TargetEpoch:         1,
			TargetHash:          chainhash.HashH([]byte("justifiedblockhash")),
		},
		ParticipationBitfield: []uint8{0, 1, 0},
		CustodyBitfield:       []uint8{0, 1, 0},
	}

	baseAttProto := baseAtt.ToProto()
	fromProto, err := primitives.PendingAttestationFromProto(baseAttProto)
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(fromProto, baseAtt); diff != nil {
		t.Fatal(diff)
	}
}
