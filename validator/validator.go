package validator

import (
	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/pb"
)

// Validator is a single validator to keep track of
type Validator struct {
	keystore           Keystore
	blockchainRPC      pb.BlockchainRPCClient
	p2pRPC             pb.P2PRPCClient
	id                 uint32
	logger             *logrus.Entry
	mempool            *mempool
	config             *config.Config
	attestationRequest chan assignment
}

// NewValidator gets a validator
func NewValidator(keystore Keystore, blockchainRPC pb.BlockchainRPCClient, p2pRPC pb.P2PRPCClient, id uint32, mempool *mempool, c *config.Config) *Validator {
	v := &Validator{
		keystore:           keystore,
		blockchainRPC:      blockchainRPC,
		p2pRPC:             p2pRPC,
		id:                 id,
		mempool:            mempool,
		config:             c,
		attestationRequest: make(chan assignment),
	}
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	v.logger = l.WithField("validator", id)
	return v
}

// RunValidator keeps track of assignments and creates/signs attestations as needed.
func (v *Validator) RunValidator() error {
	for {
		// wait for validator manager to ask for attestation
		si := <-v.attestationRequest

		if si.isProposer {
			err := v.proposeBlock(si)
			if err != nil {
				return err
			}
		} else {
			err := v.attestBlock(si)
			if err != nil {
				return err
			}
		}
	}
}

func getAttestation(information assignment) (*primitives.AttestationData, [32]byte, error) {
	a := primitives.AttestationData{
		Slot:                information.slot,
		Shard:               information.shard,
		BeaconBlockHash:     information.beaconBlockHash,
		EpochBoundaryHash:   information.epochBoundaryRoot,
		ShardBlockHash:      chainhash.Hash{}, // only attest to 0 hashes in phase 0
		LatestCrosslinkHash: information.latestCrosslinks[information.shard].ShardBlockHash,
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

func (v *Validator) signAttestation(hashAttestation [32]byte, data primitives.AttestationData, committeeSize uint64, committeeIndex uint64) (*primitives.Attestation, error) {
	signature, err := bls.Sign(v.keystore.GetKeyForValidator(v.id), hashAttestation[:], bls.DomainAttestation)
	if err != nil {
		return nil, err
	}

	participationBitfield := make([]uint8, (committeeSize+7)/8)
	custodyBitfield := make([]uint8, (committeeSize+7)/8)
	participationBitfield[committeeIndex/8] = 1 << (committeeIndex % 8)

	att := &primitives.Attestation{
		Data:                  data,
		ParticipationBitfield: participationBitfield,
		CustodyBitfield:       custodyBitfield,
		AggregateSig:          signature.Serialize(),
	}

	return att, nil
}
