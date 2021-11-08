package primitives

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/prysmaticlabs/go-ssz"

	"github.com/phoreproject/graphene/beacon/config"
	"github.com/phoreproject/graphene/bls"
	"github.com/phoreproject/graphene/chainhash"
)

// ValidateAttestation checks if the attestation is valid.
func (s *State) ValidateAttestation(att Attestation, verifySignature bool, c *config.Config) error {
	if att.Data.TargetEpoch == s.EpochIndex {
		if att.Data.SourceEpoch != s.JustifiedEpoch {
			return fmt.Errorf("expected source epoch to equal the justified epoch if the target epoch is the current epoch (expected: %d, got %d)", s.JustifiedEpoch, att.Data.SourceEpoch)
		}

		justifiedHash, err := s.GetRecentBlockHash(s.JustifiedEpoch*c.EpochLength, c)
		if err != nil {
			return err
		}

		if !att.Data.SourceHash.IsEqual(justifiedHash) {
			return fmt.Errorf("expected source hash to equal the current epoch hash if the target epoch is the current epoch (expected: %s, got %s)", justifiedHash, att.Data.TargetHash)
		}

		if !att.Data.LatestCrosslinkHash.IsEqual(&s.LatestCrosslinks[att.Data.Shard].ShardBlockHash) {
			return fmt.Errorf("expected latest crosslink hash to match if the target epoch is the current epoch (expected: %s, got %s)",
				s.LatestCrosslinks[att.Data.Shard].ShardBlockHash,
				att.Data.LatestCrosslinkHash)
		}
	} else if att.Data.TargetEpoch == s.EpochIndex-1 {
		if att.Data.SourceEpoch != s.PreviousJustifiedEpoch {
			return fmt.Errorf("expected source epoch to equal the previous justified epoch if the target epoch is the previous epoch (expected: %d, got %d)", s.PreviousJustifiedEpoch, att.Data.SourceEpoch)
		}

		previousJustifiedHash, err := s.GetRecentBlockHash(s.PreviousJustifiedEpoch*c.EpochLength, c)
		if err != nil {
			return err
		}

		if !att.Data.SourceHash.IsEqual(previousJustifiedHash) {
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

	if verifySignature {
		participants, err := s.GetAttestationParticipants(att.Data, att.ParticipationBitfield, c)
		if err != nil {
			return err
		}

		dataRoot, err := ssz.HashTreeRoot(AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
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
			return fmt.Errorf("attestation signature is invalid. expected committee with members: %v for slot %d shard %d", participants, att.Data.Slot, att.Data.Shard)
		}
	}

	return nil
}

// applyAttestation verifies and applies an attestation to the given state.
func (s *State) applyAttestation(att Attestation, c *config.Config, verifySignature bool, proposerIndex uint32) error {
	err := s.ValidateAttestation(att, verifySignature, c)
	if err != nil {
		return err
	}

	// these checks are dependent on when the attestation is included
	if att.Data.Slot+c.MinAttestationInclusionDelay > s.Slot {
		return fmt.Errorf("attestation included too soon (expected s.Slot > %d, got %d)", att.Data.Slot+c.MinAttestationInclusionDelay, s.Slot)
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

func (s *State) validateParticipationSignature(voteHash chainhash.Hash, participation []uint8, signature [48]byte) error {
	aggregatedPublicKey := bls.NewAggregatePublicKey()

	if len(participation) != (len(s.ValidatorRegistry)+7)/8 {
		return errors.New("vote participation array incorrect length")
	}

	for i := 0; i < len(s.ValidatorRegistry); i++ {
		voted := uint8(participation[i/8])&(1<<uint(i%8)) != 0

		if voted {
			pk, err := s.ValidatorRegistry[i].GetPublicKey()
			if err != nil {
				return err
			}

			aggregatedPublicKey.AggregatePubKey(pk)
		}
	}

	sig, err := bls.DeserializeSignature(signature)
	if err != nil {
		return err
	}

	valid, err := bls.VerifySig(aggregatedPublicKey, voteHash[:], sig, bls.DomainVote)

	if err != nil {
		return err
	}

	if !valid {
		return errors.New("vote signature did not validate")
	}

	return nil
}

// applyVote validates a vote and adds it to pending votes.
func (s *State) applyVote(vote AggregatedVote, config *config.Config) error {
	voteHash, err := ssz.HashTreeRoot(vote.Data)
	if err != nil {
		return err
	}

	for i, proposal := range s.Proposals {
		proposalHash, err := ssz.HashTreeRoot(proposal.Data)
		if err != nil {
			return err
		}

		// proposal is already active
		if bytes.Equal(proposalHash[:], voteHash[:]) {
			// ignore if already queued
			if proposal.Queued {
				return nil
			}

			err := s.validateParticipationSignature(voteHash, vote.Participation, vote.Signature)
			if err != nil {
				return err
			}

			needed := len(vote.Participation) - len(s.Proposals[i].Participation)
			if needed > 0 {
				s.Proposals[i].Participation = append(s.Proposals[i].Participation, make([]uint8, needed)...)
			}

			// update the proposal
			for j := range vote.Participation {
				s.Proposals[i].Participation[j] |= vote.Participation[j]
			}

			return nil
		}
	}

	proposerBitSet := vote.Participation[vote.Data.Proposer/8] & (1 << uint(vote.Data.Proposer%8))

	if proposerBitSet == 0 {
		return errors.New("could not process vote with proposer bit not set")
	}

	if vote.Data.Type == Cancel {
		foundProposalToCancel := false

		for _, proposal := range s.Proposals {
			proposalHash, err := ssz.HashTreeRoot(proposal.Data)
			if err != nil {
				return err
			}

			if bytes.Equal(vote.Data.ActionHash[:], proposalHash[:]) {
				foundProposalToCancel = true
			}
		}

		if !foundProposalToCancel {
			return errors.New("could not find proposal to cancel")
		}
	}

	err = s.validateParticipationSignature(voteHash, vote.Participation, vote.Signature)
	if err != nil {
		return err
	}

	s.ValidatorBalances[vote.Data.Proposer] -= config.ProposalCost

	s.Proposals = append(s.Proposals, ActiveProposal{
		Data:          vote.Data,
		Participation: vote.Participation,
		StartEpoch:    s.EpochIndex,
		Queued:        false,
	})

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

	if block.BlockHeader.ValidatorIndex != proposerIndex {
		return fmt.Errorf("proposer index doesn't match (expected: %d, got %d)", proposerIndex, block.BlockHeader.ValidatorIndex)
	}

	blockWithoutSignature := block.Copy()
	blockWithoutSignature.BlockHeader.Signature = bls.EmptySignature.Serialize()
	blockWithoutSignatureRoot, err := ssz.HashTreeRoot(blockWithoutSignature)
	if err != nil {
		return err
	}

	proposal := ProposalSignedData{
		Slot:      s.Slot,
		Shard:     con.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot, err := ssz.HashTreeRoot(proposal)
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

	randaoRevealSerialized, err := ssz.HashTreeRoot(block.BlockHeader.RandaoReveal)
	if err != nil {
		return err
	}

	for i := range s.NextRandaoMix {
		s.NextRandaoMix[i] ^= randaoRevealSerialized[i]
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
		err := s.applyAttestation(a, con, verifySignature, proposerIndex)
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

	for _, v := range block.BlockBody.Votes {
		err := s.applyVote(v, con)
		if err != nil {
			return err
		}
	}

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
