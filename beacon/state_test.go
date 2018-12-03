package beacon_test

import (
	"fmt"
	"testing"

	"github.com/phoreproject/synapse/beacon/internal/util"
	"github.com/phoreproject/synapse/bls"

	"github.com/phoreproject/synapse/transaction"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/beacon/primitives"
)

func TestLastBlockOnInitialSetup(t *testing.T) {
	b, err := util.SetupBlockchain(blockchain.MainNetConfig.ShardCount*blockchain.MainNetConfig.MinCommitteeSize*2+1, &blockchain.MainNetConfig)
	if err != nil {
		t.Fatal(err)
	}

	b0, err := b.LastBlock()
	if err != nil {
		t.Fatal(err)
	}

	if b0.SlotNumber != 0 {
		t.Fatal("invalid last block for initial chain")
	}

	_, err = util.MineBlockWithFullAttestations(b)
	if err != nil {
		t.Fatal(err)
	}

	if b.Height() != 1 {
		t.Fatalf("invalid height (expected: 1, got: %d)", b.Height())
	}

	b1, err := b.LastBlock()
	if err != nil {
		t.Fatal(err)
	}

	if b1.SlotNumber != 1 {
		t.Fatal("invalid last block after mining 1 block")
	}
}

func TestStateInitialization(t *testing.T) {
	b, err := util.SetupBlockchain(blockchain.MainNetConfig.ShardCount*blockchain.MainNetConfig.MinCommitteeSize*2+1, &blockchain.MainNetConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = util.MineBlockWithFullAttestations(b)
	if err != nil {
		t.Fatal(err)
	}

	s := b.GetState()

	if len(s.ShardAndCommitteeForSlots[0]) == 0 {
		t.Errorf("invalid initial validator entries")
	}

	if len(s.ShardAndCommitteeForSlots) != blockchain.MainNetConfig.CycleLength*2 {
		t.Errorf("shardandcommitteeforslots array is not big enough (got: %d, expected: %d)", len(s.ShardAndCommitteeForSlots), blockchain.MainNetConfig.CycleLength)
	}
}

/*
func TestCrystallizedStateCopy(t *testing.T) {
	c := blockchain.CrystallizedState{
		Crosslinks: []primitives.Crosslink{
			{
				RecentlyChanged: false,
				Slot:            0,
				Hash:            zeroHash,
			},
		},
		Validators: []primitives.Validator{
			{
				WithdrawalShardID: 0,
			},
		},
		ShardAndCommitteeForSlots: [][]primitives.ShardAndCommittee{
			{
				{
					ShardID: 0,
				},
			},
		},
		DepositsPenalizedInPeriod: []uint64{1},
	}

	c1 := c.Copy()
	c1.Crosslinks[0] = primitives.Crosslink{
		RecentlyChanged: true,
		Slot:            1,
		Hash:            zeroHash,
	}

	c1.Validators[0] = primitives.Validator{
		WithdrawalShardID: 1,
	}

	c1.ShardAndCommitteeForSlots[0][0] = primitives.ShardAndCommittee{
		ShardID: 1,
	}

	c1.DepositsPenalizedInPeriod[0] = 2

	if c.Crosslinks[0].RecentlyChanged {
		t.Errorf("crystallized state was not copied correctly")
		return
	}

	if c.Validators[0].WithdrawalShardID == 1 {
		t.Errorf("crystallized state was not copied correctly")
		return
	}

	if c.ShardAndCommitteeForSlots[0][0].ShardID == 1 {
		t.Errorf("crystallized state was not copied correctly")
		return
	}

	if c.DepositsPenalizedInPeriod[0] == 2 {
		t.Errorf("crystallized state was not copied correctly")
		return
	}
}
*/

func TestShardCommitteeByShardID(t *testing.T) {
	committees := []primitives.ShardAndCommittee{
		{
			Shard:     0,
			Committee: []uint32{0, 1, 2, 3},
		},
		{
			Shard:     1,
			Committee: []uint32{4, 5, 6, 7},
		},
		{
			Shard:     2,
			Committee: []uint32{0, 1, 2, 3},
		},
		{
			Shard:     3,
			Committee: []uint32{99, 100, 101, 102, 103},
		},
		{
			Shard:     4,
			Committee: []uint32{0, 1, 2, 3},
		},
		{
			Shard:     5,
			Committee: []uint32{4, 5, 6, 7},
		},
	}

	s, err := blockchain.ShardCommitteeByShardID(3, committees)
	if err != nil {
		t.Fatal(err)
	}
	if s[0] != 99 {
		t.Fatal("did not find correct shard")
	}

	_, err = blockchain.ShardCommitteeByShardID(9, committees)
	if err == nil {
		t.Fatal("did not error on non-existent shard ID")
	}

	shardAndCommitteeForSlots := [][]primitives.ShardAndCommittee{committees}

	s, err = blockchain.CommitteeInShardAndSlot(0, 3, shardAndCommitteeForSlots)
	if err != nil {
		t.Fatal(err)
	}
	if s[0] != 99 {
		t.Fatal("did not find correct shard")
	}

	cState := blockchain.BeaconState{
		LastStateRecalculationSlot: 64,
		ShardAndCommitteeForSlots:  shardAndCommitteeForSlots,
	}

	att, err := cState.GetAttesterIndices(64, 3, &blockchain.MainNetConfig)
	if err != nil {
		t.Fatal(err)
	}
	if att[0] != 99 {
		t.Fatal("did not find correct shard")
	}
}

func TestAttestationValidation(t *testing.T) {
	b, err := util.SetupBlockchain(16448, &blockchain.MainNetConfig) // committee size of 257 (% 8 = 1)
	if err != nil {
		t.Fatal(err)
	}

	lb, err := b.LastBlock()
	if err != nil {
		t.Fatal(err)
	}

	assignment := b.GetState().ShardAndCommitteeForSlots[lb.SlotNumber][0]

	committeeSize := len(assignment.Committee)

	attesterBitfield := make([]byte, (len(assignment.Committee)+7)/8)

	for i := range assignment.Committee {
		attesterBitfield, _ = util.SetBit(attesterBitfield, uint32(i))
	}

	b0, err := b.GetNodeByHeight(0)
	if err != nil {
		t.Fatal(err)
	}

	att := &transaction.AttestationRecord{
		Data: transaction.AttestationSignedData{
			Slot:                       lb.SlotNumber,
			Shard:                      assignment.Shard,
			ParentHashes:               []chainhash.Hash{},
			ShardBlockHash:             chainhash.Hash{},
			LastCrosslinkHash:          chainhash.Hash{},
			ShardBlockCombinedDataRoot: chainhash.Hash{},
			JustifiedSlot:              0,
		},
		AttesterBitfield: attesterBitfield,
		PoCBitfield:      []uint8{},
		AggregateSig:     bls.Signature{},
	}

	err = b.ValidateAttestationRecord(att, lb, &blockchain.MainNetConfig)
	if err != nil {
		t.Fatal(err)
	}

	att = &transaction.AttestationRecord{
		Data: transaction.AttestationSignedData{
			Slot:                       lb.SlotNumber + 1,
			Shard:                      assignment.Shard,
			ParentHashes:               []chainhash.Hash{},
			ShardBlockHash:             chainhash.Hash{},
			LastCrosslinkHash:          chainhash.Hash{},
			ShardBlockCombinedDataRoot: chainhash.Hash{},
			JustifiedSlot:              0,
		},
		AttesterBitfield: attesterBitfield,
		PoCBitfield:      []uint8{},
		AggregateSig:     bls.Signature{},
	}

	err = b.ValidateAttestationRecord(att, lb, &blockchain.MainNetConfig)
	if err == nil {
		t.Fatal("did not catch slot number being too high")
	}

	att = &transaction.AttestationRecord{
		Data: transaction.AttestationSignedData{
			Slot:                       lb.SlotNumber,
			Shard:                      assignment.Shard,
			ParentHashes:               []chainhash.Hash{},
			ShardBlockHash:             b0,
			LastCrosslinkHash:          chainhash.Hash{},
			ShardBlockCombinedDataRoot: chainhash.Hash{},
			JustifiedSlot:              10,
		},
		AttesterBitfield: attesterBitfield,
		PoCBitfield:      []uint8{},
		AggregateSig:     bls.Signature{},
	}

	err = b.ValidateAttestationRecord(att, lb, &blockchain.MainNetConfig)
	if err == nil {
		t.Fatal("did not catch slot number being out of bounds")
	}

	att = &transaction.AttestationRecord{
		Data: transaction.AttestationSignedData{
			Slot:                       lb.SlotNumber,
			Shard:                      100,
			ParentHashes:               []chainhash.Hash{},
			ShardBlockHash:             b0,
			LastCrosslinkHash:          chainhash.Hash{},
			ShardBlockCombinedDataRoot: chainhash.Hash{},
			JustifiedSlot:              0,
		},
		AttesterBitfield: attesterBitfield,
		PoCBitfield:      []uint8{},
		AggregateSig:     bls.Signature{},
	}

	err = b.ValidateAttestationRecord(att, lb, &blockchain.MainNetConfig)
	if err == nil {
		t.Fatal("did not catch invalid shard ID")
	}

	att = &transaction.AttestationRecord{
		Data: transaction.AttestationSignedData{
			Slot:                       lb.SlotNumber,
			Shard:                      assignment.Shard,
			ParentHashes:               []chainhash.Hash{},
			ShardBlockHash:             b0,
			LastCrosslinkHash:          chainhash.Hash{},
			ShardBlockCombinedDataRoot: chainhash.Hash{},
			JustifiedSlot:              0,
		},
		AttesterBitfield: append(attesterBitfield, byte(0x00)),
		PoCBitfield:      []uint8{},
		AggregateSig:     bls.Signature{},
	}

	err = b.ValidateAttestationRecord(att, lb, &blockchain.MainNetConfig)
	if err == nil {
		t.Fatal("did not catch invalid attester bitfield (too many bytes)")
	}

	for bitToSet := uint32(0); bitToSet <= 6; bitToSet++ {

		// t.Logf("attester bitfield length: %d, max validators: %d, validator: %d\n", len(attesterBitfield), len(attesterBitfield)*8, uint32(committeeSize)+bitToSet)

		newAttesterBitField := make([]byte, len(attesterBitfield))
		copy(newAttesterBitField, attesterBitfield)

		modifiedAttesterBitfield, err := util.SetBit(newAttesterBitField, uint32(committeeSize)+bitToSet)
		if err != nil {
			t.Fatal(err)
		}

		att = &transaction.AttestationRecord{
			Data: transaction.AttestationSignedData{
				Slot:                       lb.SlotNumber,
				Shard:                      assignment.Shard,
				ParentHashes:               []chainhash.Hash{},
				ShardBlockHash:             b0,
				LastCrosslinkHash:          chainhash.Hash{},
				ShardBlockCombinedDataRoot: chainhash.Hash{},
				JustifiedSlot:              0,
			},
			AttesterBitfield: modifiedAttesterBitfield,
			PoCBitfield:      []uint8{},
			AggregateSig:     bls.Signature{},
		}

		err = b.ValidateAttestationRecord(att, lb, &blockchain.MainNetConfig)
		if err == nil {
			t.Fatal("did not catch invalid attester bitfield (not enough 0s at end)")
		}
		// t.Log(err)
	}
}

func TestCrystallizedStateTransition(t *testing.T) {
	b, err := util.SetupBlockchain(blockchain.MainNetConfig.ShardCount*blockchain.MainNetConfig.MinCommitteeSize*2+5, &blockchain.MainNetConfig)
	if err != nil {
		t.Fatal(err)
	}

	firstValidator := b.GetState().ShardAndCommitteeForSlots[0][0].Committee[0]

	for i := uint64(0); i < uint64(b.GetConfig().CycleLength)+b.GetConfig().MinimumValidatorSetChangeInterval; i++ {
		blk, err := util.MineBlockWithFullAttestations(b)
		if err != nil {
			t.Error(err)
		}
		for _, a := range blk.Attestations {
			fmt.Printf("block %d including shard attestation %d\n", a.Data.Slot, a.Data.Shard)
		}
	}

	firstValidator2 := b.GetState().ShardAndCommitteeForSlots[0][0].Committee[0]
	if firstValidator == firstValidator2 {
		t.Fatal("validators were not shuffled")
	}
}
