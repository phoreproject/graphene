package validator

import (
	"context"

	"github.com/inconshreveable/log15"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
)

// Validator is a single validator to keep track of
type Validator struct {
	secretKey *bls.SecretKey
	rpc       pb.BlockchainRPCClient
	id        uint32
	slot      uint64
	shard     uint32
	proposer  bool
	logger    log15.Logger
	newSlot   chan uint64
	newCycle  chan bool
}

// NewValidator gets a validator
func NewValidator(key *bls.SecretKey, rpc pb.BlockchainRPCClient, id uint32, newSlot chan uint64, newCycle chan bool) *Validator {
	return &Validator{secretKey: key, rpc: rpc, id: id, logger: log15.New("validator", id), newSlot: newSlot, newCycle: newCycle}
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
	v.logger.Info("running validator", "validator", v.id)

	err := v.GetSlotAssignment()
	if err != nil {
		return err
	}

	v.logger.Info("got slot assignment", "slot", v.slot, "shard", v.shard, "proposer?", v.proposer)

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
	v.logger.Debug("proposing block", "block", v.slot, "shard", v.shard)
	return nil
}

func (v *Validator) attestBlock() error {
	v.logger.Debug("attesting block", "block", v.slot, "shard", v.shard)
	return nil
}
