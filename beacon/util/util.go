package util

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/phoreproject/synapse/validator"

	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// SetupBlockchain sets up a blockchain with a certain number of initial validators
func SetupBlockchain(initialValidators int, c *config.Config) (*beacon.Blockchain, validator.Keystore, error) {
	return SetupBlockchainWithTime(initialValidators, c, time.Now())
}

// SetupBlockchainWithTime sets up a blockchain with a certain number of initial validators and genesis time
func SetupBlockchainWithTime(initialValidators int, c *config.Config, genesisTime time.Time) (*beacon.Blockchain, validator.Keystore, error) {
	keystore := validator.NewFakeKeyStore()

	var validators []beacon.InitialValidatorEntry

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
			DepositSize:           c.MaxDeposit,
		})
	}

	b, err := beacon.NewBlockchainWithInitialValidators(db.NewInMemoryDB(), c, validators, true, uint64(genesisTime.Unix()))
	if err != nil {
		return nil, nil, err
	}

	return b, &keystore, nil
}

// MineBlockWithSpecialsAndAttestations mines a block with the given specials and attestations.
func MineBlockWithSpecialsAndAttestations(b *beacon.Blockchain, attestations []primitives.Attestation, proposerSlashings []primitives.ProposerSlashing, casperSlashings []primitives.CasperSlashing, deposits []primitives.Deposit, exits []primitives.Exit, k validator.Keystore, proposerIndex uint32) (*primitives.Block, error) {
	parentRoot := b.View.Chain.Tip().Hash

	stateRoot := b.View.Chain.Tip().StateRoot

	slotNumber := b.View.Chain.Tip().Slot + 1

	var slotsBytes [8]byte
	binary.BigEndian.PutUint64(slotsBytes[:], slotNumber)
	slotBytesHash := chainhash.HashH(slotsBytes[:])

	randaoSig, err := bls.Sign(k.GetKeyForValidator(proposerIndex), slotBytesHash[:], bls.DomainRandao)
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

	_, _, err = b.ProcessBlock(&block1, false, true)
	if err != nil {
		return nil, err
	}

	return &block1, nil
}

// GenerateFakeAttestations generates a bunch of fake attestations.
func GenerateFakeAttestations(s *primitives.State, b *beacon.Blockchain, keys validator.Keystore) ([]primitives.Attestation, error) {
	if s.Slot == 0 {
		return []primitives.Attestation{}, nil
	}

	lastSlot := b.View.Chain.Tip().Slot

	if lastSlot == 0 {
		return []primitives.Attestation{}, nil
	}

	config := b.GetConfig()

	assignments, err := s.GetShardCommitteesAtSlot(lastSlot-1, config)
	if err != nil {
		return nil, err
	}

	attestations := make([]primitives.Attestation, len(assignments))

	for i, assignment := range assignments {
		epochIndex := lastSlot / config.EpochLength

		targetHash, err := b.View.Chain.GetBlockBySlot(epochIndex * config.EpochLength)
		if err != nil {
			return nil, err
		}

		justifiedEpoch := s.JustifiedEpoch
		crosslinks := s.LatestCrosslinks
		if lastSlot%config.EpochLength == 0 {
			justifiedEpoch = s.PreviousJustifiedEpoch

			targetHash, err = b.View.Chain.GetBlockBySlot(epochIndex*config.EpochLength - config.EpochLength)
			if err != nil {
				return nil, err
			}

			epochIndex--

			crosslinks = s.PreviousCrosslinks
		}

		justifiedNode, err := b.View.Chain.GetBlockBySlot(justifiedEpoch * config.EpochLength)
		if err != nil {
			return nil, err
		}

		dataToSign := primitives.AttestationData{
			Slot:                lastSlot,
			Shard:               assignment.Shard,
			BeaconBlockHash:     b.View.Chain.Tip().Hash,
			SourceEpoch:         justifiedEpoch,
			SourceHash:          justifiedNode.Hash,
			ShardBlockHash:      chainhash.Hash{},
			LatestCrosslinkHash: crosslinks[assignment.Shard].ShardBlockHash,
			TargetEpoch:         epochIndex,
			TargetHash:          targetHash.Hash,
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

	bitfield[id/8] = bitfield[id/8] | (1 << (id % 8))

	return bitfield, nil
}

// MineBlockWithFullAttestations generates attestations to include in a block and mines it.
func MineBlockWithFullAttestations(b *beacon.Blockchain, keystore validator.Keystore, proposerIndex uint32) (*primitives.Block, error) {
	state, err := b.GetUpdatedState(b.View.Chain.Tip().Slot + 1)
	if err != nil {
		return nil, err
	}

	atts, err := GenerateFakeAttestations(state, b, keystore)
	if err != nil {
		return nil, err
	}

	return MineBlockWithSpecialsAndAttestations(b, atts, []primitives.ProposerSlashing{}, []primitives.CasperSlashing{}, []primitives.Deposit{}, []primitives.Exit{}, keystore, proposerIndex)
}
