package util

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/blockchain"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/db"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/serialization"
	"github.com/phoreproject/synapse/transaction"
)

var randaoSecret = chainhash.HashH([]byte("randao"))
var zeroHash = chainhash.Hash{}

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

func GenerateFakeAttestations(b *blockchain.Blockchain) ([]transaction.Attestation, error) {
	lb, err := b.LastBlock()
	if err != nil {
		return nil, err
	}

	assignments := b.GetShardsAndCommitteesForSlot(lb.SlotNumber)

	attestations := make([]transaction.Attestation, len(assignments))

	for i, assignment := range assignments {

		attesterBitfield := make([]byte, (len(assignment.Committee)+7)/8)

		for i := range assignment.Committee {
			attesterBitfield, _ = SetBit(attesterBitfield, uint32(i))
		}

		slotHash, err := b.GetNodeByHeight(b.GetState().Crystallized.LastJustifiedSlot)
		if err != nil {
			return nil, err
		}

		attestations[i] = transaction.Attestation{
			Slot:                lb.SlotNumber,
			ShardID:             assignment.ShardID,
			JustifiedSlot:       b.GetState().Crystallized.LastJustifiedSlot,
			JustifiedBlockHash:  slotHash,
			ObliqueParentHashes: []chainhash.Hash{},
			AttesterBitField:    attesterBitfield,
			AggregateSignature:  bls.Signature{},
			ShardBlockHash:      chainhash.HashH([]byte(fmt.Sprintf("shard %d slot %d", assignment.ShardID, lb.SlotNumber))),
		}
	}

	return attestations, nil
}

func SetBit(bitfield []byte, id uint32) ([]byte, error) {
	if uint32(len(bitfield)*8) < id {
		return nil, fmt.Errorf("bitfield is too short")
	}

	bitfield[id/8] = bitfield[id/8] | (128 >> (id % 8))

	return bitfield, nil
}

func MineBlockWithFullAttestations(b *blockchain.Blockchain) (*primitives.Block, error) {
	atts, err := GenerateFakeAttestations(b)
	if err != nil {
		return nil, err
	}

	return MineBlockWithSpecialsAndAttestations(b, []transaction.Transaction{}, atts)
}
