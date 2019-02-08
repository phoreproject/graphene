package util

import (
	"fmt"

	"github.com/phoreproject/prysm/shared/ssz"

	"github.com/golang/protobuf/proto"

	"github.com/phoreproject/synapse/validator"

	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/beacon/primitives"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
)

var randaoSecret = chainhash.HashH([]byte("randao"))

var pocSecret = chainhash.HashH([]byte("poc"))
var zeroHash = chainhash.Hash{}

// SetupBlockchain sets up a blockchain with a certain number of initial validators
func SetupBlockchain(initialValidators int, config *config.Config) (*beacon.Blockchain, validator.Keystore, error) {
	keystore := validator.FakeKeyStore{}

	validators := []beacon.InitialValidatorEntry{}

	for i := 0; i <= initialValidators; i++ {
		key := keystore.GetKeyForValidator(uint32(i))
		pub := key.DerivePublicKey()
		// fmt.Println(pub)
		hashPub := pub.Hash()
		proofOfPossession, err := bls.Sign(key, hashPub, bls.DomainDeposit)
		if err != nil {
			return nil, nil, err
		}
		validators = append(validators, beacon.InitialValidatorEntry{
			PubKey:                *pub,
			ProofOfPossession:     *proofOfPossession,
			WithdrawalShard:       1,
			WithdrawalCredentials: chainhash.Hash{},
			DepositSize:           32,
		})
	}

	b, err := beacon.NewBlockchainWithInitialValidators(db.NewInMemoryDB(), config, validators)
	if err != nil {
		return nil, nil, err
	}

	return b, &keystore, nil
}

// MineBlockWithSpecialsAndAttestations mines a block with the given specials and attestations.
func MineBlockWithSpecialsAndAttestations(b *beacon.Blockchain, specials []transaction.Transaction, attestations []transaction.AttestationRecord) (*primitives.Block, error) {
	lastBlock, err := b.LastBlock()
	if err != nil {
		return nil, err
	}

	parentRoot, err := ssz.TreeHash(lastBlock)
	if err != nil {
		return nil, err
	}

	stateRoot, err := ssz.TreeHash(lastBlock)
	if err != nil {
		return nil, err
	}

	state := b.GetState()
	proposerIndex := state.GetBeaconProposerIndex(lastBlock.BlockHeader.SlotNumber+1, b.GetConfig())

	block1 := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber: lastBlock.BlockHeader.SlotNumber + 1,
			ParentRoot: parentRoot,
			StateRoot:  stateRoot,
		},
		BlockBody: primitives.BlockBody{},
	}

	err = b.ProcessBlock(&block1)
	if err != nil {
		return nil, err
	}

	return &block1, nil
}

// GenerateFakeAttestations generates a bunch of fake attestations.
func GenerateFakeAttestations(b *beacon.Blockchain, keys validator.Keystore) ([]transaction.AttestationRecord, error) {
	lb, err := b.LastBlock()
	if err != nil {
		return nil, err
	}

	assignments := b.GetShardsAndCommitteesForSlot(lb.SlotNumber)

	attestations := make([]transaction.AttestationRecord, len(assignments))

	for i, assignment := range assignments {
		slotHash, err := b.GetNodeByHeight(b.GetState().JustificationSource)
		if err != nil {
			return nil, err
		}

		dataToSign := transaction.AttestationSignedData{
			Slot:                       lb.SlotNumber,
			Shard:                      assignment.Shard,
			ParentHashes:               []chainhash.Hash{},
			ShardBlockHash:             chainhash.HashH([]byte(fmt.Sprintf("shard %d slot %d", assignment.Shard, lb.SlotNumber))),
			LastCrosslinkHash:          chainhash.Hash{},
			ShardBlockCombinedDataRoot: slotHash,
			JustifiedSlot:              b.GetState().JustificationSource,
		}

		data, _ := proto.Marshal(dataToSign.ToProto())

		attesterBitfield := make([]byte, (len(assignment.Committee)+7)/8)
		aggregateSig := bls.NewAggregateSignature()

		for i, n := range assignment.Committee {
			attesterBitfield, _ = SetBit(attesterBitfield, uint32(i))
			key := keys.GetKeyForValidator(n)
			sig, err := bls.Sign(key, data)
			if err != nil {
				return nil, err
			}
			aggregateSig.AggregateSig(sig)
		}

		attestations[i] = transaction.AttestationRecord{
			Data:             dataToSign,
			AttesterBitfield: attesterBitfield,
			PoCBitfield:      make([]uint8, 32),
			AggregateSig:     *aggregateSig,
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
func MineBlockWithFullAttestations(b *beacon.Blockchain, keystore validator.Keystore) (*primitives.Block, error) {
	atts, err := GenerateFakeAttestations(b, keystore)
	if err != nil {
		return nil, err
	}

	return MineBlockWithSpecialsAndAttestations(b, []transaction.Transaction{}, atts)
}
