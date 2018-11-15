package blockchain_test

import (
	"fmt"
	"testing"

	"github.com/phoreproject/synapse/blockchain/util"
	"github.com/phoreproject/synapse/bls"

	"github.com/phoreproject/synapse/transaction"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/blockchain"
	"github.com/phoreproject/synapse/primitives"
)

func TestLastBlockOnInitialSetup(t *testing.T) {
	b, err := util.SetupBlockchain()
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
	b, err := util.SetupBlockchain()
	if err != nil {
		t.Fatal(err)
	}

	_, err = util.MineBlockWithFullAttestations(b)
	if err != nil {
		t.Fatal(err)
	}

	s := b.GetState()

	if len(s.Crystallized.ShardAndCommitteeForSlots[0]) == 0 {
		t.Errorf("invalid initial validator entries")
	}

	if len(s.Crystallized.ShardAndCommitteeForSlots) != blockchain.MainNetConfig.CycleLength*2 {
		t.Errorf("shardandcommitteeforslots array is not big enough (got: %d, expected: %d)", len(s.Crystallized.ShardAndCommitteeForSlots), blockchain.MainNetConfig.CycleLength)
	}
}

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

func TestShardCommitteeByShardID(t *testing.T) {
	committees := []primitives.ShardAndCommittee{
		{
			ShardID:   0,
			Committee: []uint32{0, 1, 2, 3},
		},
		{
			ShardID:   1,
			Committee: []uint32{4, 5, 6, 7},
		},
		{
			ShardID:   2,
			Committee: []uint32{0, 1, 2, 3},
		},
		{
			ShardID:   3,
			Committee: []uint32{99, 100, 101, 102, 103},
		},
		{
			ShardID:   4,
			Committee: []uint32{0, 1, 2, 3},
		},
		{
			ShardID:   5,
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
		t.Fatal("did not error on non-existant shard ID")
	}

	shardAndCommitteeForSlots := [][]primitives.ShardAndCommittee{committees}

	s, err = blockchain.CommitteeInShardAndSlot(0, 3, shardAndCommitteeForSlots)
	if err != nil {
		t.Fatal(err)
	}
	if s[0] != 99 {
		t.Fatal("did not find correct shard")
	}

	cState := blockchain.CrystallizedState{
		LastStateRecalculation:    64,
		ShardAndCommitteeForSlots: shardAndCommitteeForSlots,
	}

	att, err := cState.GetAttesterIndices(&transaction.Attestation{
		Slot:    64,
		ShardID: 3,
	}, &blockchain.MainNetConfig)
	if err != nil {
		t.Fatal(err)
	}
	if att[0] != 99 {
		t.Fatal("did not find correct shard")
	}
}

func TestAttestationValidation(t *testing.T) {
	b, err := util.SetupBlockchain()
	if err != nil {
		t.Fatal(err)
	}

	lb, err := b.LastBlock()
	if err != nil {
		t.Fatal(err)
	}

	assignment := b.GetState().Crystallized.ShardAndCommitteeForSlots[lb.SlotNumber][0]

	attesterBitfield := make([]byte, (len(assignment.Committee)+7)/8)

	for i := range assignment.Committee {
		attesterBitfield, _ = util.SetBit(attesterBitfield, uint32(i))
	}

	b0, err := b.GetNodeByHeight(0)
	if err != nil {
		t.Fatal(err)
	}

	att := &transaction.Attestation{
		Slot:                lb.SlotNumber,
		ShardID:             assignment.ShardID,
		JustifiedSlot:       0,
		JustifiedBlockHash:  b0,
		ObliqueParentHashes: []chainhash.Hash{},
		AttesterBitField:    attesterBitfield,
		AggregateSignature:  bls.Signature{},
	}

	err = b.ValidateAttestation(att, lb, &blockchain.MainNetConfig)
	if err != nil {
		t.Fatal(err)
	}

	att = &transaction.Attestation{
		Slot:                lb.SlotNumber + 1,
		ShardID:             assignment.ShardID,
		JustifiedSlot:       0,
		JustifiedBlockHash:  b0,
		ObliqueParentHashes: []chainhash.Hash{},
		AttesterBitField:    attesterBitfield,
		AggregateSignature:  bls.Signature{},
	}

	err = b.ValidateAttestation(att, lb, &blockchain.MainNetConfig)
	if err == nil {
		t.Fatal("did not catch slot number being too high")
	}

	att = &transaction.Attestation{
		Slot:                lb.SlotNumber,
		ShardID:             assignment.ShardID,
		JustifiedSlot:       10,
		JustifiedBlockHash:  b0,
		ObliqueParentHashes: []chainhash.Hash{},
		AttesterBitField:    attesterBitfield,
		AggregateSignature:  bls.Signature{},
	}

	err = b.ValidateAttestation(att, lb, &blockchain.MainNetConfig)
	if err == nil {
		t.Fatal("did not catch slot number being out of bounds")
	}

	att = &transaction.Attestation{
		Slot:                lb.SlotNumber,
		ShardID:             100,
		JustifiedSlot:       0,
		JustifiedBlockHash:  b0,
		ObliqueParentHashes: []chainhash.Hash{},
		AttesterBitField:    attesterBitfield,
		AggregateSignature:  bls.Signature{},
	}

	err = b.ValidateAttestation(att, lb, &blockchain.MainNetConfig)
	if err == nil {
		t.Fatal("did not catch invalid shard ID")
	}
}

func TestCrystallizedStateTransition(t *testing.T) {
	b, err := util.SetupBlockchain()
	if err != nil {
		t.Fatal(err)
	}

	firstValidator := b.GetState().Crystallized.ShardAndCommitteeForSlots[0][0].Committee[0]

	for i := uint64(0); i < uint64(b.GetConfig().CycleLength)+b.GetConfig().MinimumValidatorSetChangeInterval; i++ {
		blk, err := util.MineBlockWithFullAttestations(b)
		if err != nil {
			t.Error(err)
		}
		for _, a := range blk.Attestations {
			fmt.Printf("block %d including shard attestation %d\n", a.Slot, a.ShardID)
		}
	}

	firstValidator2 := b.GetState().Crystallized.ShardAndCommitteeForSlots[0][0].Committee[0]
	if firstValidator == firstValidator2 {
		t.Fatal("validators were not shuffled")
	}
}
