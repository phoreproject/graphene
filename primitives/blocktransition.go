package primitives

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

// ValidateAttestation checks if the attestation is valid.
func (s *State) ValidateAttestation(att Attestation, verifySignature bool, view BlockView, c *config.Config) error {
	if att.Data.TargetEpoch == s.EpochIndex {
		if att.Data.SourceEpoch != s.JustifiedEpoch {
			return fmt.Errorf("expected source epoch to equal the justified epoch if the target epoch is the current epoch (expected: %d, got %d)", s.EpochIndex, att.Data.TargetEpoch)
		}

		justifiedHash, err := view.GetHashBySlot(s.JustifiedEpoch * c.EpochLength)
		if err != nil {
			return err
		}

		if !att.Data.SourceHash.IsEqual(&justifiedHash) {
			return fmt.Errorf("expected source hash to equal the current epoch hash if the target epoch is the current epoch (expected: %s, got %s)", justifiedHash, att.Data.TargetHash)
		}

		if !att.Data.LatestCrosslinkHash.IsEqual(&s.LatestCrosslinks[att.Data.Shard].ShardBlockHash) {
			return fmt.Errorf("expected latest crosslink hash to match if the target epoch is the current epoch (expected: %s, got %s)",
				s.LatestCrosslinks[att.Data.Shard].ShardBlockHash,
				att.Data.LatestCrosslinkHash)
		}
	} else if att.Data.TargetEpoch == s.EpochIndex-1 {
		if att.Data.SourceEpoch != s.PreviousJustifiedEpoch {
			return fmt.Errorf("expected source epoch to equal the previous justified epoch if the target epoch is the previous epoch (expected: %d, got %d)", s.EpochIndex-1, att.Data.TargetEpoch)
		}

		previousJustifiedHash, err := view.GetHashBySlot(s.PreviousJustifiedEpoch * c.EpochLength)
		if err != nil {
			return err
		}

		if !att.Data.SourceHash.IsEqual(&previousJustifiedHash) {
			return fmt.Errorf("expected source hash to equal the previous justified hash if the target epoch is the previous epoch (expected: %s, got %s)", previousJustifiedHash, att.Data.TargetHash)
		}

		if !att.Data.LatestCrosslinkHash.IsEqual(&s.PreviousCrosslinks[att.Data.Shard].ShardBlockHash) {
			return fmt.Errorf("expected latest crosslink hash to match if the target epoch is the previous epoch(expected: %s, got %s)",
				s.PreviousCrosslinks[att.Data.Shard].ShardBlockHash,
				att.Data.LatestCrosslinkHash)
		}
	} else {
		return fmt.Errorf("attestation should have target epoch of either the current epoch (%d) or the previous epoch (%d) but got %d", s.EpochIndex, s.EpochIndex-1, att.Data.TargetEpoch)
	}

	if len(s.LatestCrosslinks) <= int(att.Data.Shard) {
		return errors.New("invalid shard number")
	}

	latestCrosslinkRoot := s.LatestCrosslinks[att.Data.Shard].ShardBlockHash

	if !att.Data.LatestCrosslinkHash.IsEqual(&latestCrosslinkRoot) && !att.Data.ShardBlockHash.IsEqual(&latestCrosslinkRoot) {
		return errors.New("latest crosslink is invalid")
	}

	if verifySignature {
		participants, err := s.GetAttestationParticipants(att.Data, att.ParticipationBitfield, c)
		if err != nil {
			return err
		}

		dataRoot, err := ssz.TreeHash(AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
		if err != nil {
			return err
		}

		groupPublicKey := bls.NewAggregatePublicKey()
		for _, p := range participants {
			pub, err := s.ValidatorRegistry[p].GetPublicKey()
			if err != nil {
				return err
			}
			groupPublicKey.AggregatePubKey(pub)
		}

		aggSig, err := bls.DeserializeSignature(att.AggregateSig)
		if err != nil {
			return err
		}

		valid, err := bls.VerifySig(groupPublicKey, dataRoot[:], aggSig, GetDomain(s.ForkData, att.Data.Slot, bls.DomainAttestation))
		if err != nil {
			return err
		}

		if !valid {
			return errors.New("attestation signature is invalid")
		}
	}

	node, err := view.GetHashBySlot(att.Data.Slot)
	if err != nil {
		return err
	}

	if !att.Data.BeaconBlockHash.IsEqual(&node) {
		return fmt.Errorf("beacon block hash is invalid (expected: %s, got: %s)", node, att.Data.BeaconBlockHash)
	}

	if !att.Data.ShardBlockHash.IsEqual(&zeroHash) {
		return errors.New("invalid block Hash")
	}

	return nil
}

// applyAttestation verifies and applies an attestation to the given state.
func (s *State) applyAttestation(att Attestation, c *config.Config, view BlockView, verifySignature bool, proposerIndex uint32) error {
	err := s.ValidateAttestation(att, verifySignature, view, c)
	if err != nil {
		return err
	}

	// these checks are dependent on when the attestation is included
	if att.Data.Slot+c.MinAttestationInclusionDelay > s.Slot {
		return errors.New("attestation included too soon")
	}

	// 4 -> 8 should not work
	// 5 -> 8 should work
	if att.Data.Slot+c.EpochLength <= s.Slot {
		return errors.New("attestation was not included within 1 epoch")
	}

	if (att.Data.Slot-1)/c.EpochLength != att.Data.TargetEpoch {
		return errors.New("attestation slot did not match target epoch")
	}

	if att.Data.TargetEpoch == s.EpochIndex {
		s.CurrentEpochAttestations = append(s.CurrentEpochAttestations, PendingAttestation{
			Data:                  att.Data,
			ParticipationBitfield: att.ParticipationBitfield,
			CustodyBitfield:       att.CustodyBitfield,
			InclusionDelay:        s.Slot - att.Data.Slot,
			ProposerIndex:         proposerIndex,
		})
	} else {
		s.PreviousEpochAttestations = append(s.PreviousEpochAttestations, PendingAttestation{
			Data:                  att.Data,
			ParticipationBitfield: att.ParticipationBitfield,
			CustodyBitfield:       att.CustodyBitfield,
			InclusionDelay:        s.Slot - att.Data.Slot,
			ProposerIndex:         proposerIndex,
		})
	}

	return nil
}

// ProcessBlock tries to apply a block to the state.
func (s *State) ProcessBlock(block *Block, con *config.Config, view BlockView, verifySignature bool) error {
	proposerIndex, err := s.GetBeaconProposerIndex(block.BlockHeader.SlotNumber-1, con)
	if err != nil {
		return err
	}

	if block.BlockHeader.SlotNumber != s.Slot {
		return fmt.Errorf("block has incorrect slot number (expecting: %d, got: %d)", s.Slot, block.BlockHeader.SlotNumber)
	}

	blockWithoutSignature := block.Copy()
	blockWithoutSignature.BlockHeader.Signature = bls.EmptySignature.Serialize()
	blockWithoutSignatureRoot, err := ssz.TreeHash(blockWithoutSignature)
	if err != nil {
		return err
	}

	proposal := ProposalSignedData{
		Slot:      s.Slot,
		Shard:     con.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot, err := ssz.TreeHash(proposal)
	if err != nil {
		return err
	}

	proposerPub, err := s.ValidatorRegistry[proposerIndex].GetPublicKey()
	if err != nil {
		return err
	}

	proposerSig, err := bls.DeserializeSignature(block.BlockHeader.Signature)
	if err != nil {
		return err
	}

	// process block and randao verifications concurrently

	if verifySignature {
		verificationResult := make(chan error)

		go func() {
			valid, err := bls.VerifySig(proposerPub, proposalRoot[:], proposerSig, bls.DomainProposal)
			if err != nil {
				verificationResult <- err
			}

			if !valid {
				verificationResult <- fmt.Errorf("block had invalid signature (expected signature from validator %d)", proposerIndex)
			}

			verificationResult <- nil
		}()

		var slotBytes [8]byte
		binary.BigEndian.PutUint64(slotBytes[:], block.BlockHeader.SlotNumber)
		slotBytesHash := chainhash.HashH(slotBytes[:])

		randaoSig, err := bls.DeserializeSignature(block.BlockHeader.RandaoReveal)
		if err != nil {
			return err
		}

		go func() {
			valid, err := bls.VerifySig(proposerPub, slotBytesHash[:], randaoSig, bls.DomainRandao)
			if err != nil {
				verificationResult <- err
			}
			if !valid {
				verificationResult <- errors.New("block has invalid randao signature")
			}

			verificationResult <- nil
		}()

		result1 := <-verificationResult
		result2 := <-verificationResult

		if result1 != nil {
			return result1
		}

		if result2 != nil {
			return result2
		}
	}

	randaoRevealSerialized, err := ssz.TreeHash(block.BlockHeader.RandaoReveal)
	if err != nil {
		return err
	}

	for i := range s.RandaoMix {
		s.RandaoMix[i] ^= randaoRevealSerialized[i]
	}

	if len(block.BlockBody.ProposerSlashings) > con.MaxProposerSlashings {
		return errors.New("more than maximum proposer slashings")
	}

	if len(block.BlockBody.CasperSlashings) > con.MaxCasperSlashings {
		return errors.New("more than maximum casper slashings")
	}

	if len(block.BlockBody.Attestations) > con.MaxAttestations {
		return errors.New("more than maximum attestations")
	}

	if len(block.BlockBody.Exits) > con.MaxExits {
		return errors.New("more than maximum exits")
	}

	if len(block.BlockBody.Deposits) > con.MaxDeposits {
		return errors.New("more than maximum deposits")
	}

	if len(block.BlockBody.Votes) > con.MaxVotes {
		return errors.New("more than maximum votes")
	}

	for _, slashing := range block.BlockBody.ProposerSlashings {
		err := s.applyProposerSlashing(slashing, con)
		if err != nil {
			return err
		}
	}

	for _, c := range block.BlockBody.CasperSlashings {
		err := s.applyCasperSlashing(c, con)
		if err != nil {
			return err
		}
	}

	for _, a := range block.BlockBody.Attestations {
		err := s.applyAttestation(a, con, view, verifySignature, proposerIndex)
		if err != nil {
			return err
		}
	}

	// process deposits here

	for _, e := range block.BlockBody.Exits {
		err := s.ApplyExit(e, con)
		if err != nil {
			return err
		}
	}

	//blockTransitionTime := time.Since(blockTransitionStart)

	//logrus.WithField("slot", s.Slot).WithField("block", block.BlockHeader.SlotNumber).WithField("duration", blockTransitionTime).Info("block transition")

	// Check state root.
	expectedStateRoot, err := view.GetLastStateRoot()
	if err != nil {
		return err
	}
	if !block.BlockHeader.StateRoot.IsEqual(&expectedStateRoot) {
		return fmt.Errorf("state root doesn't match (expected: %s, got: %s)", expectedStateRoot, block.BlockHeader.StateRoot)
	}

	return nil
}
