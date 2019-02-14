package validator

import (
	"sync"

	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/sirupsen/logrus"
)

type mempool struct {
	attestationMempool attestationMempool
}

func newMempool() mempool {
	return mempool{newAttestationMempool()}
}

type attestationMempool struct {
	attestations     []primitives.Attestation
	attestationsLock *sync.RWMutex
}

func newAttestationMempool() attestationMempool {
	return attestationMempool{
		attestations:     make([]primitives.Attestation, 0),
		attestationsLock: new(sync.RWMutex),
	}
}

func (am *attestationMempool) processNewAttestation(att primitives.Attestation) {
	am.attestationsLock.Lock()
	defer am.attestationsLock.Unlock()
	// TODO: make this more efficient
	for _, a := range am.attestations {
		if a.Data.Slot == att.Data.Slot && a.Data.Shard == att.Data.Shard {
			for i, b := range a.ParticipationBitfield {
				if b&att.ParticipationBitfield[i] != 0 {
					logrus.Debug("duplicate attestation, ignoring")
					return
				}
			}
		}
	}
	am.attestations = append(am.attestations, att)
}

type attestationWithRealSig struct {
	custodyBitfield       []uint8
	participationBitfield []uint8
	aggregateSignature    *bls.Signature
	data                  primitives.AttestationData
}

func (am *attestationMempool) getAttestationsToInclude(slot uint64, c *config.Config) ([]primitives.Attestation, error) {
	// include any attestations
	aggregatedAttestations := make(map[chainhash.Hash]*attestationWithRealSig)

	am.attestationsLock.Lock()
	for _, att := range am.attestations {
		if att.Data.Slot+c.MinAttestationInclusionDelay >= slot {
			continue // don't include attestations that aren't valid yet
		}

		hash, err := ssz.TreeHash(primitives.AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
		if err != nil {
			return nil, err
		}

		sig, err := bls.DeserializeSignature(att.AggregateSig)
		if err != nil {
			return nil, err
		}

		if _, found := aggregatedAttestations[hash]; !found {
			aggregatedAttestations[hash] = &attestationWithRealSig{
				data:                  att.Data,
				participationBitfield: att.ParticipationBitfield,
				aggregateSignature:    sig,
				custodyBitfield:       make([]uint8, len(att.ParticipationBitfield)),
			}
		} else {
			aggregatedAttestations[hash].aggregateSignature.AggregateSig(sig)
			for i := range aggregatedAttestations[hash].participationBitfield {
				aggregatedAttestations[hash].participationBitfield[i] |= att.ParticipationBitfield[i]
			}
			for i := range aggregatedAttestations[hash].custodyBitfield {
				aggregatedAttestations[hash].custodyBitfield[i] |= att.CustodyBitfield[i]
			}
		}
	}
	am.attestationsLock.Unlock()

	attestationsToInclude := make([]primitives.Attestation, len(aggregatedAttestations))
	i := 0
	for _, att := range aggregatedAttestations {
		attestationsToInclude[i] = primitives.Attestation{
			AggregateSig:          att.aggregateSignature.Serialize(),
			ParticipationBitfield: att.participationBitfield,
			Data:                  att.data,
			CustodyBitfield:       att.custodyBitfield,
		}
		i++
	}

	return attestationsToInclude, nil
}

// we want to remove attestations that will always be invalid or pointless
func (am *attestationMempool) removeAttestationsBeforeSlot(slot uint64) {
	am.attestationsLock.Lock()
	defer am.attestationsLock.Unlock()
	newAttestations := make([]primitives.Attestation, 0)
	for _, att := range am.attestations {
		if att.Data.Slot >= slot {
			newAttestations = append(newAttestations, att)
		}
	}
	am.attestations = newAttestations
}

// we want to remove attestations that will always be invalid or pointless
func (am *attestationMempool) removeAttestationsFromBitfield(slot uint64, shard uint64, bitfield []uint8) {
	am.attestationsLock.Lock()
	defer am.attestationsLock.Unlock()
	newAttestations := make([]primitives.Attestation, 0)
	for _, att := range am.attestations {
		if att.Data.Slot == slot && att.Data.Shard == shard {
			intersect := false
			for i := range att.ParticipationBitfield {
				if att.ParticipationBitfield[i]&bitfield[i] != 0 {
					intersect = true
				}
			}
			if intersect {
				continue
			}
		}
		newAttestations = append(newAttestations, att)
	}
	am.attestations = newAttestations
}

func (am *attestationMempool) size() int {
	am.attestationsLock.Lock()
	defer am.attestationsLock.Unlock()
	return len(am.attestations)
}
