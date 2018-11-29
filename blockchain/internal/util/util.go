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

// SetupBlockchain sets up a blockchain with a certain number of initial validators
func SetupBlockchain(initialValidators int, config *blockchain.Config) (*blockchain.Blockchain, error) {

	randaoCommitment := chainhash.HashH(randaoSecret[:])

	validators := []blockchain.InitialValidatorEntry{}

	for i := 0; i <= initialValidators; i++ {
		validators = append(validators, blockchain.InitialValidatorEntry{
			PubKey:                bls.PublicKey{},
			ProofOfPossession:     bls.Signature{},
			WithdrawalShard:       1,
			WithdrawalCredentials: serialization.Address{},
			RandaoCommitment:      randaoCommitment,
		})
	}

	b, err := blockchain.NewBlockchainWithInitialValidators(db.NewInMemoryDB(), config, validators)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// MineBlockWithSpecialsAndAttestations mines a block with the given specials and attestations.
func MineBlockWithSpecialsAndAttestations(b *blockchain.Blockchain, specials []transaction.Transaction, attestations []transaction.AttestationRecord) (*primitives.Block, error) {
	lastBlock, err := b.LastBlock()
	if err != nil {
		return nil, err
	}

	block1 := primitives.Block{
		SlotNumber:            lastBlock.SlotNumber + 1,
		RandaoReveal:          chainhash.HashH([]byte(fmt.Sprintf("test test %d", lastBlock.SlotNumber))),
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

// GenerateFakeAttestations generates a bunch of fake attestations.
func GenerateFakeAttestations(b *blockchain.Blockchain) ([]transaction.AttestationRecord, error) {
	lb, err := b.LastBlock()
	if err != nil {
		return nil, err
	}

	assignments := b.GetShardsAndCommitteesForSlot(lb.SlotNumber)

	attestations := make([]transaction.AttestationRecord, len(assignments))

	for i, assignment := range assignments {

		attesterBitfield := make([]byte, (len(assignment.Committee)+7)/8)

		for i := range assignment.Committee {
			attesterBitfield, _ = SetBit(attesterBitfield, uint32(i))
		}

		slotHash, err := b.GetNodeByHeight(b.GetState().JustificationSource)
		if err != nil {
			return nil, err
		}

		attestations[i] = transaction.AttestationRecord{
			Data: transaction.AttestationSignedData{
				Slot:                       lb.SlotNumber,
				Shard:                      assignment.Shard,
				ParentHashes:               []chainhash.Hash{},
				ShardBlockHash:             chainhash.HashH([]byte(fmt.Sprintf("shard %d slot %d", assignment.Shard, lb.SlotNumber))),
				LastCrosslinkHash:          chainhash.Hash{},
				ShardBlockCombinedDataRoot: slotHash,
				JustifiedSlot:              b.GetState().JustificationSource,
			},
			AttesterBitfield: attesterBitfield,
			PoCBitfield:      make([]uint8, 32),
			AggregateSig:     0,
		}
	}

	return attestations, nil
}

// SetBit sets a bit in a bitfield.
func SetBit(bitfield []byte, id uint32) ([]byte, error) {
	if uint32(len(bitfield)*8) <= id {
		return nil, fmt.Errorf("bitfield is too short")
	}

	bitfield[id/8] = bitfield[id/8] | (128 >> (id % 8))

	return bitfield, nil
}

// MineBlockWithFullAttestations generates attestations to include in a block and mines it.
func MineBlockWithFullAttestations(b *blockchain.Blockchain) (*primitives.Block, error) {
	atts, err := GenerateFakeAttestations(b)
	if err != nil {
		return nil, err
	}

	return MineBlockWithSpecialsAndAttestations(b, []transaction.Transaction{}, atts)
}
