package validator

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/golang/protobuf/proto"
	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
)

const maxAttemptsAttestation = 10

func hammingWeight(b uint8) int {
	b = b - ((b >> 1) & 0x55)
	b = (b & 0x33) + ((b >> 2) & 0x33)
	return int(((b + (b >> 4)) & 0x0F) * 0x01)
}

// ListenForMessages listens for incoming attestations and sends them to a channel.
func (v *Validator) ListenForMessages(topic string, newAttestations chan *primitives.Attestation, errReturn chan error) uint64 {
	sub, err := v.p2pRPC.Subscribe(context.Background(), &pb.SubscriptionRequest{
		Topic: topic,
	})
	if err != nil {
		errReturn <- err
	}
	messages, err := v.p2pRPC.ListenForMessages(context.Background(), sub)
	if err != nil {
		errReturn <- err
	}
	go func() {
		for {
			msg, err := messages.Recv()
			if err != nil {
				errReturn <- err
			}

			attRaw := msg.Data

			var incomingAttestationProto pb.Attestation

			err = proto.Unmarshal(attRaw, &incomingAttestationProto)
			if err != nil {
				v.logger.WithField("err", err).Debug("received invalid attestation")
				continue
			}

			incomingAttestation, err := primitives.AttestationFromProto(&incomingAttestationProto)
			if err != nil {
				v.logger.WithField("err", err).Debug("received invalid attestation")
				continue
			}

			newAttestations <- incomingAttestation
		}
	}()

	return sub.ID
}

func (v *Validator) proposeBlock(information slotInformation) error {
	newAttestations := make(chan *primitives.Attestation)

	errReturn := make(chan error)

	go func() {
		attData, hashAttestation, err := v.getAttestation(information)
		if err != nil {
			errReturn <- err
		}

		att, err := v.signAttestation(hashAttestation, *attData)
		if err != nil {
			errReturn <- err
		}
		newAttestations <- att
	}()

	topic := fmt.Sprintf("signedAttestation %d", information.slot+1)

	// listen for
	subID := v.ListenForMessages(topic, newAttestations, errReturn)

	defer func() {
		v.p2pRPC.Unsubscribe(context.Background(), &pb.Subscription{ID: subID})
	}()

	attestations, err := v.mempool.attestationMempool.getAttestationsToInclude(v.slot, v.config)
	if err != nil {
		return err
	}

	v.logger.Debug("creating block")

	parentRootBytes, err := v.blockchainRPC.GetBlockHash(context.Background(), &pb.GetBlockHashRequest{SlotNumber: v.slot - 1})
	if err != nil {
		return err
	}

	parentRoot, err := chainhash.NewHash(parentRootBytes.Hash)
	if err != nil {
		return err
	}

	stateRootBytes, err := v.blockchainRPC.GetStateRoot(context.Background(), &empty.Empty{})
	if err != nil {

	}

	stateRoot, err := chainhash.NewHash(stateRootBytes.StateRoot)
	if err != nil {
		return err
	}

	var proposerSlotsBytes [8]byte
	binary.BigEndian.PutUint64(proposerSlotsBytes[:], v.keystore.GetProposerSlots(v.id))

	key := v.keystore.GetKeyForValidator(v.id)

	randaoSig, err := bls.Sign(key, proposerSlotsBytes[:], bls.DomainRandao)
	if err != nil {
		return err
	}

	v.logger.WithFields(logrus.Fields{
		"mempoolSize": v.mempool.attestationMempool.size(),
		"including":   len(attestations),
	}).Debug("getting some mempool transactions")

	newBlock := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   v.slot,
			ParentRoot:   *parentRoot,
			StateRoot:    *stateRoot,
			RandaoReveal: randaoSig.Serialize(),
			Signature:    bls.EmptySignature.Serialize(),
		},
		BlockBody: primitives.BlockBody{
			Attestations:      attestations,
			ProposerSlashings: []primitives.ProposerSlashing{},
			CasperSlashings:   []primitives.CasperSlashing{},
			Deposits:          []primitives.Deposit{},
			Exits:             []primitives.Exit{},
		},
	}

	blockHash, err := ssz.TreeHash(newBlock)
	if err != nil {
		return err
	}

	v.logger.WithField("blockHash", fmt.Sprintf("%x", blockHash)).Info("signing block")

	psd := primitives.ProposalSignedData{
		Slot:      v.slot,
		Shard:     config.MainNetConfig.BeaconShardNumber,
		BlockHash: blockHash,
	}

	psdHash, err := ssz.TreeHash(psd)
	if err != nil {
		return err
	}

	sig, err := bls.Sign(v.keystore.GetKeyForValidator(v.id), psdHash[:], bls.DomainProposal)
	if err != nil {
		return err
	}
	newBlock.BlockHeader.Signature = sig.Serialize()

	v.logger.Debug("submitting block")

	submitBlockRequest := &pb.SubmitBlockRequest{
		Block: newBlock.ToProto(),
	}

	_, err = v.blockchainRPC.SubmitBlock(context.Background(), submitBlockRequest)

	v.logger.Debug("submitted block")

	for _, a := range attestations {
		v.mempool.attestationMempool.removeAttestationsFromBitfield(a.Data.Slot, a.Data.Shard, a.ParticipationBitfield)
	}

	return err
}
