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
	keystore      Keystore
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
func NewValidator(keystore Keystore, blockchainRPC pb.BlockchainRPCClient, p2pRPC pb.P2PRPCClient, id uint32, newSlot chan slotInformation, newCycle chan bool) *Validator {
	v := &Validator{
		keystore:      keystore,
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

		// if next block is our turn
		if slotInformation.slot+1 == v.slot {
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
	signature, err := bls.Sign(v.keystore.GetKeyForValidator(v.id), hashAttestation[:], bls.DomainAttestation)
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
