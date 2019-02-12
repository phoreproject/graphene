package validator

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/phoreproject/synapse/beacon/config"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/golang/protobuf/proto"
	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/sirupsen/logrus"
)

const maxAttemptsAttestation = 10

type committeeAggregation struct {
	custodyBitfield       []uint8
	participationBitfield []uint8
	aggregateSignature    *bls.Signature
	data                  primitives.AttestationData
}

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

	aggregations, err := v.listenForAttestations(information, newAttestations)
	if err != nil {
		return err
	}

	v.logger.WithField("attestations", len(aggregations)).Debug("collected all attestations for block")

	attestations := make([]primitives.Attestation, len(aggregations))

	i := 0

	for _, a := range aggregations {
		attestations[i].AggregateSig = a.aggregateSignature.Serialize()
		attestations[i].CustodyBitfield = a.custodyBitfield
		attestations[i].ParticipationBitfield = a.participationBitfield
		attestations[i].Data = a.data
		i++
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

	fmt.Println(newBlock.BlockHeader)

	blockHash, err := ssz.TreeHash(newBlock)
	if err != nil {
		return err
	}

	fmt.Println(blockHash)

	v.logger.WithField("blockHash", blockHash).Debug("signing block")

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

	return err
}

// ListenForAttestations listens for attestations to add to our proposal.
func (v *Validator) listenForAttestations(information slotInformation, newAttestations chan *primitives.Attestation) (map[chainhash.Hash]*committeeAggregation, error) {
	// we should request all of the shards assigned to this slot and then re-request as needed
	committees, err := v.blockchainRPC.GetCommitteesForSlot(context.Background(), &pb.GetCommitteesForSlotRequest{Slot: information.slot + 1})
	if err != nil {
		return nil, err
	}

	currentAttestationBitfieldsTotal := make(map[uint64][]byte)

	committeeAttested := make(map[uint32]int)
	committeeTotal := make(map[uint32]int)

	shards := make(map[uint32]*pb.ShardCommittee)
	for _, c := range committees.Committees {
		shards[uint32(c.Shard)] = c
		committeeAttested[uint32(c.Shard)] = 0
		committeeTotal[uint32(c.Shard)] = len(c.Committee)
	}

	for s, c := range shards {
		fmt.Println("setting up shard", s, information.slot+1)

		currentAttestationBitfieldsTotal[uint64(s)] = make([]byte, (len(c.Committee)+7)/8)

		requestTopic := fmt.Sprintf("signedAttestation request %d %d", information.slot+1, s)

		attRequest := &pb.AttestationRequest{
			ParticipationBitfield: make([]byte, len(c.Committee)),
		}

		attRequestBytes, err := proto.Marshal(attRequest)
		if err != nil {
			return nil, err
		}

		v.p2pRPC.Broadcast(context.Background(), &pb.MessageAndTopic{
			Topic: requestTopic,
			Data:  attRequestBytes,
		})
	}

	currentAttestations := make(map[chainhash.Hash]*committeeAggregation)

	t := time.NewTicker(time.Second * 3)

	attempt := 0

	for {
		select {
		case newAtt := <-newAttestations:
			// first we should check to make sure that the intersection between the bitfields is the empty set
			signedHash, err := ssz.TreeHash(primitives.AttestationDataAndCustodyBit{Data: newAtt.Data, PoCBit: false})
			if err != nil {
				return nil, err
			}

			// if we don't have a running aggregate attestation for the signedHash, create one
			if _, found := currentAttestations[signedHash]; !found {
				committee, err := v.GetCommittee(v.slot, uint32(newAtt.Data.Shard))
				if err != nil {
					return nil, err
				}

				currentAttestations[signedHash] = &committeeAggregation{
					data:                  newAtt.Data,
					participationBitfield: make([]uint8, (len(committee)+7)/8),
					custodyBitfield:       make([]uint8, (len(committee)+7)/8),
					aggregateSignature:    bls.NewAggregateSignature(),
				}
			}

			if len(currentAttestationBitfieldsTotal[newAtt.Data.Shard]) != len(newAtt.ParticipationBitfield) {
				v.logger.WithFields(logrus.Fields{
					"numExpected": len(currentAttestationBitfieldsTotal[newAtt.Data.Shard]),
					"numReceived": len(newAtt.ParticipationBitfield),
					"shard":       newAtt.Data.Shard,
				}).Debug("attestation participation bitfield doesn't match committee size")
				continue
			}

			for i := range newAtt.ParticipationBitfield {
				if currentAttestationBitfieldsTotal[newAtt.Data.Shard][i]&newAtt.ParticipationBitfield[i] != 0 {
					v.logger.Debug("received duplicate attestation")
					continue
				}
			}

			for i := range newAtt.ParticipationBitfield {
				oldBits := currentAttestations[signedHash].participationBitfield[i]
				newBits := newAtt.ParticipationBitfield[i]
				numNewBits := hammingWeight((oldBits ^ newBits) & newBits)
				currentAttestations[signedHash].participationBitfield[i] |= newAtt.ParticipationBitfield[i]
				committeeAttested[uint32(newAtt.Data.Shard)] += numNewBits
			}

			sig, err := bls.DeserializeSignature(newAtt.AggregateSig)
			if err != nil {
				return nil, err
			}
			currentAttestations[signedHash].aggregateSignature.AggregateSig(sig)

			shouldBreak := true

			for i := range committeeAttested {
				v.logger.WithFields(logrus.Fields{
					"shard":       i,
					"numAttested": committeeAttested[i],
					"total":       committeeTotal[i],
				}).Debug("received attestation")
				if committeeAttested[i] != committeeTotal[i] {
					shouldBreak = false
				}
			}

			if shouldBreak {
				return currentAttestations, nil
			}
		case <-t.C:
			if attempt > maxAttemptsAttestation {
				return currentAttestations, nil
			}
			attempt++

			for _, a := range currentAttestations {
				for i := range currentAttestationBitfieldsTotal[a.data.Shard] {
					currentAttestationBitfieldsTotal[a.data.Shard][i] |= a.participationBitfield[i]
				}
			}

			for shard, committeeBitfield := range currentAttestationBitfieldsTotal {
				requestTopic := fmt.Sprintf("signedAttestation request %d %d", information.slot+1, shard)

				attRequest := &pb.AttestationRequest{
					ParticipationBitfield: committeeBitfield,
				}

				attRequestBytes, err := proto.Marshal(attRequest)
				if err != nil {
					return nil, err
				}

				v.p2pRPC.Broadcast(context.Background(), &pb.MessageAndTopic{
					Topic: requestTopic,
					Data:  attRequestBytes,
				})
			}
		}
	}
}
