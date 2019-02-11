package validator

import (
	"context"

	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
)

// Validator is a single validator to keep track of
type Validator struct {
	secretKey   *bls.SecretKey
	rpc         pb.BlockchainRPCClient
	id          uint32
	slot        uint64
	shard       uint32
	committee   []uint32
	committeeID uint32
	proposer    bool
	logger      *logrus.Logger
	newSlot     chan slotInformation
	newCycle    chan bool
}

// NewValidator gets a validator
func NewValidator(key *bls.SecretKey, rpc pb.BlockchainRPCClient, id uint32, newSlot chan slotInformation, newCycle chan bool) *Validator {
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

// GetCommittee gets the indices that the validator is a part of.
func (v *Validator) GetCommittee() error {
	validators, err := v.rpc.GetCommitteeValidatorIndices(context.Background(), &pb.GetCommitteeValidatorsRequest{
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
				err = v.proposeBlock()
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

func (v *Validator) proposeBlock() error {
	v.logger.WithField("block", v.slot).WithField("shard", v.shard).Debug("proposing block")
	return nil
}

func (v *Validator) attestBlock(information slotInformation) error {
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
		return err
	}

	signature, err := bls.Sign(v.secretKey, hashAttestation[:], bls.DomainAttestation)
	if err != nil {
		return err
	}

	participationBitfield := make([]uint8, (len(v.committee)+7)/8)
	custodyBitfield := make([]uint8, (len(v.committee)+7)/8)
	participationBitfield[v.committeeID/8] = 1 << (v.committeeID % 8)

	att := primitives.Attestation{
		Data:                  a,
		ParticipationBitfield: participationBitfield,
		CustodyBitfield:       custodyBitfield,
		AggregateSig:          signature.Serialize(),
	}

	att.Copy() // TODO: remove this

	v.logger.WithField("block", v.slot).WithField("shard", v.shard).WithField("attestationHash", hashAttestation).Debug("attesting block")
	return nil
}
