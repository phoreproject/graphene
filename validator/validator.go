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
func (v *Validator) GetCommittee() error {
	validators, err := v.blockchainRPC.GetCommitteeValidatorIndices(context.Background(), &pb.GetCommitteeValidatorsRequest{
		SlotNumber: v.slot,
		Shard:      v.shard,
	})
	if err != nil {
		return err
	}

	v.committee = validators.Validators

	for i, index := range v.committee {
		if index == v.id {
			v.committeeID = uint32(i)
		}
	}

	return nil
}

// RunValidator keeps track of assignments and creates/signs attestations as needed.
func (v *Validator) RunValidator() error {
	err := v.GetSlotAssignment()
	if err != nil {
		return err
	}

	err = v.GetCommittee()
	if err != nil {
		return err
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

			err = v.GetCommittee()
			if err != nil {
				return err
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

func (v *Validator) proposeBlock(information slotInformation) error {
	newAttestations := make(chan *primitives.Attestation)

	errReturn := make(chan error)

	attData, hashAttestation, err := v.getAttestation(information)
	if err != nil {
		return err
	}

	aggSig := bls.NewAggregateSignature()
	if err != nil {
		return err
	}

	go func() {
		att, err := v.signAttestation(hashAttestation, *attData)
		if err != nil {
			errReturn <- err
		}
		newAttestations <- att
	}()

	topic := fmt.Sprintf("signedAttestation %x", hashAttestation)

	go func() {
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

	bitfieldTotal := make([]uint8, (len(v.committee)+7)/8)

	numAttesters := 0

	for {
		select {
		case newAtt := <-newAttestations:
			// first we should check to make sure that the intersection between the bitfields is
			for i := range newAtt.ParticipationBitfield {
				if bitfieldTotal[i]&newAtt.ParticipationBitfield[i] != 0 {
					v.logger.Debug("received duplicate attestation")
					continue
				}
			}
			for i := range newAtt.ParticipationBitfield {
				bitfieldTotal[i] |= newAtt.ParticipationBitfield[i]
				numAttesters += hammingWeight(newAtt.ParticipationBitfield[i])
			}

			sig, err := bls.DeserializeSignature(newAtt.AggregateSig)
			if err != nil {
				return err
			}
			aggSig.AggregateSig(sig)

			v.logger.WithField("totalAttesters", len(v.committee)).WithField("numAttested", numAttesters).Debug("received attestation")
		case err := <-errReturn:
			return err
		}
	}
}

func (v *Validator) attestBlock(information slotInformation) error {
	attData, hash, err := v.getAttestation(information)
	if err != nil {
		return err
	}

	att, err := v.signAttestation(hash, *attData)
	if err != nil {
		return err
	}

	attestationAndPoCBit := primitives.AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false}
	hashAttestation, err := ssz.TreeHash(attestationAndPoCBit)
	if err != nil {
		return err
	}

	attProto := att.ToProto()

	attBytes, err := proto.Marshal(attProto)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("signedAttestation %x", hashAttestation)

	// v.logger.WithField("topic", topic).WithField("length", len(attBytes)).Debug("broadcasting signed attestation")

	timer := time.NewTimer(time.Second * 4)
	<-timer.C

	_, err = v.p2pRPC.Broadcast(context.Background(), &pb.MessageAndTopic{
		Topic: topic,
		Data:  attBytes,
	})
	if err != nil {
		return err
	}

	// v.logger.WithField("block", v.slot).WithField("shard", v.shard).WithField("attestationHash", fmt.Sprintf("%x", hashAttestation)).Debug("attesting block")
	return nil
}
