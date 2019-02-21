package util

import (
	"encoding/binary"
	"fmt"

	"github.com/phoreproject/prysm/shared/ssz"

	"github.com/phoreproject/synapse/validator"

	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

var randaoSecret = chainhash.HashH([]byte("randao"))

var pocSecret = chainhash.HashH([]byte("poc"))
var zeroHash = chainhash.Hash{}

// SetupBlockchain sets up a blockchain with a certain number of initial validators
func SetupBlockchain(initialValidators int, c *config.Config) (*beacon.Blockchain, validator.Keystore, error) {
	keystore := validator.NewFakeKeyStore()

	validators := []beacon.InitialValidatorEntry{}

	for i := 0; i <= initialValidators; i++ {
		key := keystore.GetKeyForValidator(uint32(i))
		pub := key.DerivePublicKey()
		hashPub, err := ssz.TreeHash(pub.Serialize())
		if err != nil {
			return nil, nil, err
		}
		proofOfPossession, err := bls.Sign(key, hashPub[:], bls.DomainDeposit)
		if err != nil {
			return nil, nil, err
		}
		validators = append(validators, beacon.InitialValidatorEntry{
			PubKey:                pub.Serialize(),
			ProofOfPossession:     proofOfPossession.Serialize(),
			WithdrawalShard:       1,
			WithdrawalCredentials: chainhash.Hash{},
			DepositSize:           c.MaxDeposit * config.UnitInCoin,
		})
	}

	b, err := beacon.NewBlockchainWithInitialValidators(db.NewInMemoryDB(), c, validators, true)
	if err != nil {
		return nil, nil, err
	}

	return b, &keystore, nil
}

// MineBlockWithSpecialsAndAttestations mines a block with the given specials and attestations.
func MineBlockWithSpecialsAndAttestations(b *beacon.Blockchain, attestations []primitives.Attestation, proposerSlashings []primitives.ProposerSlashing, casperSlashings []primitives.CasperSlashing, deposits []primitives.Deposit, exits []primitives.Exit, k validator.Keystore) (*primitives.Block, error) {
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

	slotNumber := lastBlock.BlockHeader.SlotNumber + 1

	proposerIndex, err := state.GetBeaconProposerIndex(slotNumber-1, slotNumber-1, b.GetConfig())
	if err != nil {
		return nil, err
	}

	k.IncrementProposerSlots(proposerIndex)

	var proposerSlotsBytes [8]byte
	binary.BigEndian.PutUint64(proposerSlotsBytes[:], k.GetProposerSlots(proposerIndex))

	randaoSig, err := bls.Sign(k.GetKeyForValidator(proposerIndex), proposerSlotsBytes[:], bls.DomainRandao)
	if err != nil {
		return nil, err
	}

	block1 := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotNumber,
			ParentRoot:   parentRoot,
			StateRoot:    stateRoot,
			RandaoReveal: randaoSig.Serialize(),
			Signature:    bls.EmptySignature.Serialize(),
		},
		BlockBody: primitives.BlockBody{
			Attestations:      attestations,
			ProposerSlashings: proposerSlashings,
			CasperSlashings:   casperSlashings,
			Deposits:          deposits,
			Exits:             exits,
		},
	}

	blockHash, err := ssz.TreeHash(block1)
	if err != nil {
		return nil, err
	}

	psd := primitives.ProposalSignedData{
		Slot:      slotNumber,
		Shard:     b.GetConfig().BeaconShardNumber,
		BlockHash: blockHash,
	}

	psdHash, err := ssz.TreeHash(psd)
	if err != nil {
		return nil, err
	}

	sig, err := bls.Sign(k.GetKeyForValidator(proposerIndex), psdHash[:], bls.DomainProposal)
	if err != nil {
		return nil, err
	}
	block1.BlockHeader.Signature = sig.Serialize()

	err = b.ProcessBlock(&block1)
	if err != nil {
		return nil, err
	}

	return &block1, nil
}

// GenerateFakeAttestations generates a bunch of fake attestations.
func GenerateFakeAttestations(b *beacon.Blockchain, keys validator.Keystore) ([]primitives.Attestation, error) {
	lb, err := b.LastBlock()
	if err != nil {
		return nil, err
	}

	s := b.GetState()
	assignments, err := s.GetShardCommitteesAtSlot(s.Slot, lb.BlockHeader.SlotNumber, b.GetConfig())
	if err != nil {
		return nil, err
	}

	attestations := make([]primitives.Attestation, len(assignments))

	for i, assignment := range assignments {
		epochBoundaryHash, err := b.GetEpochBoundaryHash()
		if err != nil {
			return nil, err
		}

		nextSlot := s.Slot + 1

		justifiedSlot := s.JustifiedSlot
		if lb.BlockHeader.SlotNumber < nextSlot-(nextSlot%b.GetConfig().EpochLength) {
			justifiedSlot = s.PreviousJustifiedSlot
		}

		justifiedHash, err := b.GetHashByHeight(justifiedSlot)
		if err != nil {
			return nil, err
		}

		dataToSign := primitives.AttestationData{
			Slot:                lb.BlockHeader.SlotNumber,
			Shard:               assignment.Shard,
			BeaconBlockHash:     b.Tip(),
			EpochBoundaryHash:   epochBoundaryHash,
			ShardBlockHash:      chainhash.Hash{},
			LatestCrosslinkHash: s.LatestCrosslinks[assignment.Shard].ShardBlockHash,
			JustifiedBlockHash:  justifiedHash,
			JustifiedSlot:       justifiedSlot,
		}

		dataAndCustodyBit := primitives.AttestationDataAndCustodyBit{Data: dataToSign, PoCBit: false}

		dataRoot, err := ssz.TreeHash(dataAndCustodyBit)
		if err != nil {
			return nil, err
		}

		attesterBitfield := make([]byte, (len(assignment.Committee)+7)/8)
		aggregateSig := bls.NewAggregateSignature()

		for i, n := range assignment.Committee {
			attesterBitfield, _ = SetBit(attesterBitfield, uint32(i))
			key := keys.GetKeyForValidator(n)
			sig, err := bls.Sign(key, dataRoot[:], bls.DomainAttestation)
			if err != nil {
				return nil, err
			}
			aggregateSig.AggregateSig(sig)
		}

		attestations[i] = primitives.Attestation{
			Data:                  dataToSign,
			ParticipationBitfield: attesterBitfield,
			CustodyBitfield:       make([]uint8, 32),
			AggregateSig:          aggregateSig.Serialize(),
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

	return MineBlockWithSpecialsAndAttestations(b, atts, []primitives.ProposerSlashing{}, []primitives.CasperSlashing{}, []primitives.Deposit{}, []primitives.Exit{}, keystore)
}
