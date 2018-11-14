package blockchain_test

import (
	"fmt"
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

func TestLastBlockOnInitialSetup(t *testing.T) {
	b, err := SetupBlockchain()
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

	_, err = MineBlock(b)
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

func GenerateNextBlock(b *blockchain.Blockchain, specials []transaction.Transaction, attestations []transaction.Attestation) (primitives.Block, error) {
	lb, err := b.LastBlock()
	if err != nil {
		return primitives.Block{}, err
	}

	return primitives.Block{
		SlotNumber:            lb.SlotNumber + 1,
		RandaoReveal:          chainhash.HashH([]byte("randao")),
		AncestorHashes:        blockchain.UpdateAncestorHashes(lb.AncestorHashes, lb.SlotNumber, lb.Hash()),
		ActiveStateRoot:       chainhash.Hash{},
		CrystallizedStateRoot: chainhash.Hash{},
		Specials:              specials,
		Attestations:          attestations,
	}, nil
}

var randaoSecret = chainhash.HashH([]byte("randao"))

func SetupBlockchain() (*blockchain.Blockchain, error) {

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
		return nil, err
	}

	return b, nil
}

func MineBlockWithSpecialsAndAttestations(b *blockchain.Blockchain, specials []transaction.Transaction, attestations []transaction.Attestation) (*primitives.Block, error) {
	lastBlock, err := b.LastBlock()
	if err != nil {
		return nil, err
	}

	block1 := primitives.Block{
		SlotNumber:            lastBlock.SlotNumber + 1,
		RandaoReveal:          randaoSecret,
		AncestorHashes:        blockchain.UpdateAncestorHashes(lastBlock.AncestorHashes, lastBlock.SlotNumber, lastBlock.Hash()),
		ActiveStateRoot:       zeroHash,
		CrystallizedStateRoot: zeroHash,
		Specials:              specials,
		Attestations:          attestations,
	}

	err = b.ProcessBlock(&block1)
	if err != nil {
		return nil, err
	}

	return &block1, nil
}

func MineBlock(b *blockchain.Blockchain) (*primitives.Block, error) {
	return MineBlockWithSpecialsAndAttestations(b, []transaction.Transaction{}, []transaction.Attestation{})
}

func GenerateFakeAttestations(b *blockchain.Blockchain) ([]transaction.Attestation, error) {
	lb, err := b.LastBlock()
	if err != nil {
		return nil, err
	}

	assignments := b.GetState().Crystallized.ShardAndCommitteeForSlots[lb.SlotNumber]

	attestations := make([]transaction.Attestation, len(assignments))

	for i, assignment := range assignments {

		attesterBitfield := make([]byte, (len(assignment.Committee)+7)/8)

		for i := range assignment.Committee {
			attesterBitfield, _ = SetBit(attesterBitfield, uint32(i))
		}

		attestations[i] = transaction.Attestation{
			Slot:                lb.SlotNumber,
			ShardID:             assignment.ShardID,
			JustifiedSlot:       lb.SlotNumber,
			JustifiedBlockHash:  lb.Hash(),
			ObliqueParentHashes: []chainhash.Hash{},
			AttesterBitField:    attesterBitfield,
			AggregateSignature:  bls.Signature{},
		}
	}

	return attestations, nil
}

func MineBlockWithFullAttestations(b *blockchain.Blockchain) (*primitives.Block, error) {
	atts, err := GenerateFakeAttestations(b)
	if err != nil {
		return nil, err
	}

	return MineBlockWithSpecialsAndAttestations(b, []transaction.Transaction{}, atts)
}

func TestStateInitialization(t *testing.T) {
	b, err := SetupBlockchain()
	if err != nil {
		t.Fatal(err)
	}

	_, err = MineBlock(b)
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

func SetBit(bitfield []byte, id uint32) ([]byte, error) {
	if uint32(len(bitfield)*8) < id {
		return nil, fmt.Errorf("bitfield is too short")
	}

	bitfield[id/8] = bitfield[id/8] | (128 >> (id % 8))

	return bitfield, nil
}

func TestAttestationValidation(t *testing.T) {
	b, err := SetupBlockchain()
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
		attesterBitfield, _ = SetBit(attesterBitfield, uint32(i))
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
	b, err := SetupBlockchain()
	if err != nil {
		t.Fatal(err)
	}

	MineBlockWithFullAttestations(b)
	MineBlockWithFullAttestations(b)
}
