package validator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/phoreproject/synapse/beacon/primitives"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/transaction"
	"github.com/sirupsen/logrus"
)

type validatorAndID struct {
	ID        int32
	validator primitives.Validator
}

// Validator is a single validator to keep track of
type Validator struct {
	secretKey *bls.SecretKey
	rpc       pb.BlockchainRPCClient
	p2p       pb.P2PRPCClient
	ctx       context.Context
	id        uint32
	slot      uint64
	shard     uint32
	committee []validatorAndID
	proposer  bool
	logger    *logrus.Logger
	newSlot   chan uint64
	newCycle  chan bool
}

// NewValidator gets a validator
func NewValidator(ctx context.Context, key *bls.SecretKey, rpc pb.BlockchainRPCClient, p2p pb.P2PRPCClient, id uint32, newSlot chan uint64, newCycle chan bool) *Validator {
	return &Validator{secretKey: key, rpc: rpc, p2p: p2p, id: id, logger: logrus.New(), newSlot: newSlot, newCycle: newCycle, ctx: ctx}
}

// GetSlotAssignment receives the slot assignment from the rpc.
func (v *Validator) GetSlotAssignment() error {
	slotAssignment, err := v.rpc.GetSlotAndShardAssignment(context.Background(), &pb.GetSlotAndShardAssignmentRequest{ValidatorID: v.id})
	if err != nil {
		return err
	}

	v.shard = slotAssignment.ShardID
	v.slot = slotAssignment.Slot
	v.proposer = true
	if slotAssignment.Role == pb.Role_ATTESTER {
		v.proposer = false
	}

	validators, err := v.rpc.GetCommitteeValidators(v.ctx, &pb.GetCommitteeValidatorsRequest{SlotNumber: v.slot, Shard: v.shard})
	if err != nil {
		return err
	}

	v.committee = make([]validatorAndID, len(validators.Validators))
	for i, validator := range validators.Validators {
		val, err := primitives.ValidatorFromProto(validator)
		if err != nil {
			return err
		}
		v.committee[i] = validatorAndID{validator.ID, *val}
	}

	return nil
}

// RunValidator keeps track of assignments and creates/signs attestations as needed.
func (v *Validator) RunValidator() error {
	err := v.GetSlotAssignment()
	if err != nil {
		return err
	}

	for {
		currentSlot := <-v.newSlot

		if currentSlot == v.slot {
			var err error
			if v.proposer {
				err = v.proposeBlock()
			} else {
				err = v.attestBlock()
			}
			if err != nil {
				return err
			}

			<-v.newCycle

			err = v.GetSlotAssignment()
			if err != nil {
				return err
			}
		}
	}
}

func (v *Validator) proposeBlock() error {
	v.logger.WithField("block", v.slot).WithField("shard", v.shard).Debug("proposing block")
	return nil
}

func getRightmostBitSet(i uint8) int {
	switch i - (i & (i - 1)) {
	case 1 << 0:
		return 0
	case 1 << 1:
		return 1
	case 1 << 2:
		return 2
	case 1 << 3:
		return 3
	case 1 << 4:
		return 4
	case 1 << 5:
		return 5
	case 1 << 6:
		return 6
	case 1 << 7:
		return 7
	}
	return -1
}

func (v *Validator) attestBlock() error {
	topicID := fmt.Sprintf("attestation shard %d slot %d", v.shard, v.slot)
	sub, err := v.p2p.Subscribe(v.ctx, &pb.SubscriptionRequest{Topic: topicID})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(v.ctx, 300*time.Second)

	msgStream, err := v.p2p.ListenForMessages(ctx, sub)
	if err != nil {
		cancel()
		return err
	}
	sigs := 0
	total := len(v.committee)

	// newAttestestation := &transaction.AttestationRecord{
	// 	Data: transaction.AttestationSignedData{
	// 		Slot: v.slot,
	// 		Shard: v.shard,
	// 		ParentHashes: v.
	// 	}
	// }

	// aggSig := bls.NewAggregateSignature()

	for sigs*3 < total*2 {
		newMsg, err := msgStream.Recv()
		if ctx.Err() == context.DeadlineExceeded {
			cancel()
			return errors.New("could not gather attestations in time")
		}
		if err != nil {
			cancel()
			return err
		}

		var attRecordProto *pb.AttestationRecord

		proto.Unmarshal(newMsg.Data, attRecordProto)

		attRecord, err := transaction.NewAttestationRecordFromProto(attRecordProto)
		if err != nil {
			cancel()
			return err
		}

		positionInCommittee := 0
		for len(attRecord.AttesterBitfield) > positionInCommittee*8 {
			if attRecord.AttesterBitfield[positionInCommittee/8] == 0 {
				positionInCommittee += 8
				continue
			}
			positionInCommittee += getRightmostBitSet(attRecord.AttesterBitfield[positionInCommittee/8])
			break
		}

		// attRecord.AggregateSig
	}
	cancel()
	msgStream.Recv()
	v.logger.WithField("block", v.slot).WithField("shard", v.shard).Debug("attesting block")
	return nil
}
