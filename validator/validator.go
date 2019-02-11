package validator

import (
	"context"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
	"github.com/sirupsen/logrus"
)

// Validator is a single validator to keep track of
type Validator struct {
	secretKey *bls.SecretKey
	rpc       pb.BlockchainRPCClient
	id        uint32
	slot      uint64
	shard     uint32
	proposer  bool
	logger    *logrus.Logger
	newSlot   chan uint64
	newCycle  chan bool
}

// NewValidator gets a validator
func NewValidator(key *bls.SecretKey, rpc pb.BlockchainRPCClient, id uint32, newSlot chan uint64, newCycle chan bool) *Validator {
	v := &Validator{secretKey: key, rpc: rpc, id: id, logger: logrus.New(), newSlot: newSlot, newCycle: newCycle}
	v.logger.SetLevel(logrus.DebugLevel)
	return v
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

func (v *Validator) attestBlock() error {
	v.logger.WithField("block", v.slot).WithField("shard", v.shard).Debug("attesting block")
	return nil
}
