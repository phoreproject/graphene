package blockchain_test

import (
	"crypto/rand"
	"testing"

	"github.com/phoreproject/synapse/bls"

	"github.com/phoreproject/synapse/serialization"
	"github.com/phoreproject/synapse/transaction"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/blockchain"
	"github.com/phoreproject/synapse/db"
	"github.com/phoreproject/synapse/primitives"
)

func generateAncestorHashes(hashes []chainhash.Hash) []chainhash.Hash {
	var out [32]chainhash.Hash

	for i := range out {
		if i <= len(hashes)-1 {
			out[i] = hashes[i]
		} else {
			out[i] = zeroHash
		}
	}

	return out[:]
}

func GenerateNextBlock(b *blockchain.Blockchain) primitives.Block {
	return primitives.Block{}
}

func TestStateInitialization(t *testing.T) {
	var randaoSecret [32]byte

	rand.Read(randaoSecret[:])

	randaoCommitment := chainhash.HashH(randaoSecret[:])

	validators := []blockchain.InitialValidatorEntry{}

	for i := 0; i <= 100000; i++ {
		validators = append(validators, blockchain.InitialValidatorEntry{
			PubKey:            bls.PublicKey{},
			ProofOfPossession: bls.Signature{},
			WithdrawalShard:   1,
			WithdrawalAddress: serialization.Address{},
			RandaoCommitment:  randaoCommitment,
		})
	}

	b, err := blockchain.NewBlockchainWithInitialValidators(db.NewInMemoryDB(), &blockchain.MainNetConfig, validators)
	if err != nil {
		t.Error(err)
		return
	}

	lastBlock, err := b.GetNodeByHeight(0)
	if err != nil {
		t.Error(err)
		return
	}

	block1 := primitives.Block{
		SlotNumber:            1,
		RandaoReveal:          zeroHash,
		AncestorHashes:        generateAncestorHashes([]chainhash.Hash{lastBlock}),
		ActiveStateRoot:       zeroHash,
		CrystallizedStateRoot: zeroHash,
		Specials:              []transaction.Transaction{},
		Attestations:          []transaction.Attestation{},
	}

	err = b.ProcessBlock(&block1)
	if err != nil {
		t.Error(err)
		return
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
				Hash:            &zeroHash,
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
		Hash:            &zeroHash,
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
