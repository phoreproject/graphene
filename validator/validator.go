package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
)

// Validator is a single validator to keep track of
type Validator struct {
	secretKey     *bls.SecretKey
	blockchainRPC pb.BlockchainRPCClient
	p2pRPC        pb.P2PRPCClient
	id            uint32
	slot          uint64
	shard         uint32
	committee     []uint32
	committeeID   uint32
	proposer      bool
	logger        *logrus.Entry
	newSlot       chan slotInformation
	newCycle      chan bool
}

// NewValidator gets a validator
func NewValidator(key *bls.SecretKey, blockchainRPC pb.BlockchainRPCClient, p2pRPC pb.P2PRPCClient, id uint32, newSlot chan slotInformation, newCycle chan bool) *Validator {
	v := &Validator{
		secretKey:     key,
		blockchainRPC: blockchainRPC,
		p2pRPC:        p2pRPC,
		id:            id,
		newSlot:       newSlot,
		newCycle:      newCycle,
	}
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	v.logger = l.WithField("validator", id)
	return v
}

// GetSlotAssignment receives the slot assignment from the rpc.
func (v *Validator) GetSlotAssignment() error {
	slotAssignment, err := v.blockchainRPC.GetSlotAndShardAssignment(context.Background(), &pb.GetSlotAndShardAssignmentRequest{ValidatorID: v.id})
	if err != nil {
		return err
	}

	v.shard = slotAssignment.ShardID
	v.slot = slotAssignment.Slot
	v.proposer = true
	if slotAssignment.Role == pb.Role_ATTESTER {
		v.proposer = false
	}

	return nil
}

// GetCommittee gets the indices that the validator is a part of.
func (v *Validator) GetCommittee(slot uint64, shard uint32) ([]uint32, error) {
	validators, err := v.blockchainRPC.GetCommitteeValidatorIndices(context.Background(), &pb.GetCommitteeValidatorsRequest{
		SlotNumber: slot,
		Shard:      shard,
	})
	if err != nil {
		return nil, err
	}

	return validators.Validators, nil
}

// RunValidator keeps track of assignments and creates/signs attestations as needed.
func (v *Validator) RunValidator() error {
	err := v.GetSlotAssignment()
	if err != nil {
		return err
	}

	committee, err := v.GetCommittee(v.slot, v.shard)
	if err != nil {
		return err
	}

	v.committee = committee

	for i, index := range committee {
		if index == v.id {
			v.committeeID = uint32(i)
		}
	}

	for {
		slotInformation := <-v.newSlot

		if slotInformation.slot == v.slot {
			var err error
			if v.proposer {
				err = v.proposeBlock(slotInformation)
			} else {
				err = v.attestBlock(slotInformation)
			}
			if err != nil {
				return err
			}

			<-v.newCycle

			err = v.GetSlotAssignment()
			if err != nil {
				return err
			}

			committee, err = v.GetCommittee(v.slot, v.shard)
			if err != nil {
				return err
			}
			v.committee = committee

			for i, index := range committee {
				if index == v.id {
					v.committeeID = uint32(i)
				}
			}
		}
	}
}

func (v *Validator) getAttestation(information slotInformation) (*primitives.AttestationData, [32]byte, error) {
	a := primitives.AttestationData{
		Slot:                information.slot,
		Shard:               uint64(v.shard),
		BeaconBlockHash:     information.beaconBlockHash,
		EpochBoundaryHash:   information.epochBoundaryRoot,
		ShardBlockHash:      chainhash.Hash{}, // only attest to 0 hashes in phase 0
		LatestCrosslinkHash: information.latestCrosslinks[v.shard].ShardBlockHash,
		JustifiedSlot:       information.justifiedSlot,
		JustifiedBlockHash:  information.justifiedRoot,
	}

	attestationAndPoCBit := primitives.AttestationDataAndCustodyBit{Data: a, PoCBit: false}
	hashAttestation, err := ssz.TreeHash(attestationAndPoCBit)
	if err != nil {
		return nil, [32]byte{}, err
	}

	return &a, hashAttestation, nil
}

func (v *Validator) signAttestation(hashAttestation [32]byte, data primitives.AttestationData) (*primitives.Attestation, error) {
	signature, err := bls.Sign(v.secretKey, hashAttestation[:], bls.DomainAttestation)
	if err != nil {
		return nil, err
	}

	participationBitfield := make([]uint8, (len(v.committee)+7)/8)
	custodyBitfield := make([]uint8, (len(v.committee)+7)/8)
	participationBitfield[v.committeeID/8] = 1 << (v.committeeID % 8)

	att := &primitives.Attestation{
		Data:                  data,
		ParticipationBitfield: participationBitfield,
		CustodyBitfield:       custodyBitfield,
		AggregateSig:          signature.Serialize(),
	}

	return att, nil
}

func hammingWeight(b uint8) int {
	b = b - ((b >> 1) & 0x55)
	b = (b & 0x33) + ((b >> 2) & 0x33)
	return int(((b + (b >> 4)) & 0x0F) * 0x01)
}

// ListenForMessages listens for incoming attestations and sends them to a channel.
func (v *Validator) ListenForMessages(topic string, newAttestations chan *primitives.Attestation, errReturn chan error) {
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
	defer func() {
		messages.CloseSend()
		v.p2pRPC.Unsubscribe(context.Background(), sub)
	}()
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
}

type committeeAggregation struct {
	bitfield           []uint8
	aggregateSignature *bls.Signature
	data               primitives.AttestationData
}

const maxAttemptsAttestation = 3

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

	topic := fmt.Sprintf("signedAttestation %d", information.slot)

	// we should request all of the shards assigned to this slot and then re-request as needed
	committees, err := v.blockchainRPC.GetCommitteesForSlot(context.Background(), &pb.GetCommitteesForSlotRequest{Slot: information.slot})
	if err != nil {
		return err
	}

	committeeAttested := make(map[uint32]int)
	committeeTotal := make(map[uint32]int)

	shards := make(map[uint32]*pb.ShardCommittee)
	for _, c := range committees.Committees {
		shards[uint32(c.Shard)] = c
		committeeAttested[uint32(c.Shard)] = 0
		committeeTotal[uint32(c.Shard)] = len(c.Committee)
	}

	currentAttestationBitfieldsTotal := make(map[uint64][]byte)

	for s, c := range shards {
		requestTopic := fmt.Sprintf("signedAttestation request %d %d", information.slot, s)

		attRequest := &pb.AttestationRequest{
			ParticipationBitfield: make([]byte, len(c.Committee)),
		}

		attRequestBytes, err := proto.Marshal(attRequest)
		if err != nil {
			return err
		}

		v.p2pRPC.Broadcast(context.Background(), &pb.MessageAndTopic{
			Topic: requestTopic,
			Data:  attRequestBytes,
		})

		currentAttestationBitfieldsTotal[uint64(s)] = make([]byte, (len(c.Committee)+7)/8)
	}

	// listen for
	go v.ListenForMessages(topic, newAttestations, errReturn)

	currentAttestations := make(map[chainhash.Hash]*committeeAggregation)

	t := time.NewTicker(time.Second * 2)

	attempt := 0

	for {
		select {
		case newAtt := <-newAttestations:
			// first we should check to make sure that the intersection between the bitfields is the empty set
			signedHash, err := ssz.TreeHash(primitives.AttestationDataAndCustodyBit{Data: newAtt.Data, PoCBit: false})
			if err != nil {
				return err
			}

			// if we don't have a running aggregate attestation for the signedHash, create one
			if _, found := currentAttestations[signedHash]; !found {
				committee, err := v.GetCommittee(v.slot, uint32(newAtt.Data.Shard))
				if err != nil {
					return err
				}

				currentAttestations[signedHash] = &committeeAggregation{
					data:               newAtt.Data,
					bitfield:           make([]uint8, (len(committee)+7)/8),
					aggregateSignature: bls.NewAggregateSignature(),
				}
			}

			if len(currentAttestations[signedHash].bitfield) != len(newAtt.ParticipationBitfield) {

				v.logger.Debug("attestation participation bitfield doesn't match committee size")
				fmt.Println(len(currentAttestations[signedHash].bitfield), len(newAtt.ParticipationBitfield))
				continue
			}

			for i := range newAtt.ParticipationBitfield {
				if currentAttestationBitfieldsTotal[newAtt.Data.Shard][i]&newAtt.ParticipationBitfield[i] != 0 {
					v.logger.Debug("received duplicate attestation")
					continue
				}
			}

			for i := range newAtt.ParticipationBitfield {
				oldBits := currentAttestations[signedHash].bitfield[i]
				newBits := newAtt.ParticipationBitfield[i]
				numNewBits := hammingWeight((oldBits ^ newBits) & newBits)
				currentAttestations[signedHash].bitfield[i] |= newAtt.ParticipationBitfield[i]
				committeeAttested[uint32(newAtt.Data.Shard)] += numNewBits
			}

			sig, err := bls.DeserializeSignature(newAtt.AggregateSig)
			if err != nil {
				return err
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
				break
			}
		case <-t.C:
			if attempt > maxAttemptsAttestation {
				break
			}
			attempt++

			for _, a := range currentAttestations {
				for i := range currentAttestationBitfieldsTotal[a.data.Shard] {
					currentAttestationBitfieldsTotal[a.data.Shard][i] |= a.bitfield[i]
				}
			}

			for shard, committeeBitfield := range currentAttestationBitfieldsTotal {
				requestTopic := fmt.Sprintf("signedAttestation request %d %d", information.slot, shard)

				attRequest := &pb.AttestationRequest{
					ParticipationBitfield: committeeBitfield,
				}

				attRequestBytes, err := proto.Marshal(attRequest)
				if err != nil {
					return err
				}

				v.p2pRPC.Broadcast(context.Background(), &pb.MessageAndTopic{
					Topic: requestTopic,
					Data:  attRequestBytes,
				})
			}
		case err := <-errReturn:
			return err
		}
	}
}

var zeroHash = [32]byte{}

func (v *Validator) attestBlock(information slotInformation) error {
	attData, hash, err := v.getAttestation(information)
	if err != nil {
		return err
	}

	att, err := v.signAttestation(hash, *attData)
	if err != nil {
		return err
	}

	attProto := att.ToProto()

	attBytes, err := proto.Marshal(attProto)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("signedAttestation %d", v.slot)

	topicRequest := fmt.Sprintf("signedAttestation request %d %d", v.slot, v.shard)

	sub, err := v.p2pRPC.Subscribe(context.Background(), &pb.SubscriptionRequest{
		Topic: topicRequest,
	})

	if err != nil {
		return err
	}

	msgListener, err := v.p2pRPC.ListenForMessages(context.Background(), sub)
	if err != nil {
		return err
	}

	defer func() {
		msgListener.CloseSend()
		v.p2pRPC.Unsubscribe(context.Background(), sub)
	}()

	for {
		msg, err := msgListener.Recv()
		if err != nil {
			return err
		}

		var attRequest pb.AttestationRequest
		err = proto.Unmarshal(msg.Data, &attRequest)
		if err != nil {
			return err
		}

		if attRequest.ParticipationBitfield[v.committeeID/8]&(1<<(v.committeeID%8)) == 0 {
			_, err = v.p2pRPC.Broadcast(context.Background(), &pb.MessageAndTopic{
				Topic: topic,
				Data:  attBytes,
			})
		} else {
			break
		}
	}
	return nil
}
