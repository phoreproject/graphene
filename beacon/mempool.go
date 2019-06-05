package beacon

import (
	"fmt"
	"sort"
	"sync"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/sirupsen/logrus"
)

// Mempool keeps track of actions (attestations, deposits, exits, slashings) to include in blocks.
type Mempool struct {
	AttestationMempool *attestationMempool
	blockchain         *Blockchain
}

// NewMempool creates a new mempool.
func NewMempool(blockchain *Blockchain) *Mempool {
	m := &Mempool{
		AttestationMempool: newAttestationMempool(blockchain),
		blockchain:         blockchain,
	}

	blockchain.RegisterNotifee(m)

	return m
}

// ConnectBlock is part of the blockchain notifee.
func (m *Mempool) ConnectBlock(b *primitives.Block) {
	for _, a := range b.BlockBody.Attestations {
		m.RemoveAttestationsFromBitfield(a.Data.Slot, a.Data.Shard, a.ParticipationBitfield)
	}
}

type attestationMempool struct {
	blockchain       *Blockchain
	attestations     []primitives.Attestation // maps the hashed data
	attestationsLock *sync.RWMutex
}

func newAttestationMempool(blockchain *Blockchain) *attestationMempool {
	return &attestationMempool{
		attestations:     make([]primitives.Attestation, 0),
		attestationsLock: new(sync.RWMutex),
		blockchain:       blockchain,
	}
}

// ProcessNewAttestation processes a new attestation to be included in a block.
func (m *Mempool) ProcessNewAttestation(att primitives.Attestation) error {
	m.AttestationMempool.attestationsLock.Lock()
	defer m.AttestationMempool.attestationsLock.Unlock()
	for _, a := range m.AttestationMempool.attestations {
		if a.Data.Slot == att.Data.Slot && a.Data.Shard == att.Data.Shard {
			for i, b := range a.ParticipationBitfield {
				if b&att.ParticipationBitfield[i] != 0 {
					logrus.Debug("duplicate attestation, ignoring")
					return nil
				}
			}
		}
	}
	m.AttestationMempool.attestations = append(m.AttestationMempool.attestations, att)

	return nil
}

type attestationWithRealSigAndCount struct {
	custodyBitfield       []uint8
	participationBitfield []uint8
	aggregateSignature    *bls.Signature
	data                  primitives.AttestationData
	count                 int
}

type byCount []attestationWithRealSigAndCount

// Len finds the length of the attestations.
func (a byCount) Len() int {
	return len(a)
}

// Swap swaps index i and j.
func (a byCount) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Less checks if index i is less than j.
func (a byCount) Less(i, j int) bool {
	return a[i].count < a[j].count
}

// GetAttestationsToInclude gets attestations to include in a block.
func (m *Mempool) GetAttestationsToInclude(slot uint64, c *config.Config) ([]primitives.Attestation, error) {
	// include any attestations
	aggregatedAttestationMap := make(map[chainhash.Hash]*attestationWithRealSigAndCount)

	am := m.AttestationMempool

	am.removeOldAttestations(slot, c)

	am.attestationsLock.Lock()

	// go through all of the (separate) attestations we've received
	for _, att := range am.attestations {
		if att.Data.Slot+c.MinAttestationInclusionDelay > slot {
			continue // don't include attestations that aren't valid yet
		}

		// get the hash of the attestation data
		hash, err := ssz.TreeHash(primitives.AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
		if err != nil {
			return nil, err
		}

		// get the signature of the attestation
		sig, err := bls.DeserializeSignature(att.AggregateSig)
		if err != nil {
			return nil, err
		}

		if _, found := aggregatedAttestationMap[hash]; !found {
			// if this isn't already included, include it
			aggregatedAttestationMap[hash] = &attestationWithRealSigAndCount{
				data:                  att.Data,
				participationBitfield: att.ParticipationBitfield,
				aggregateSignature:    sig,
				custodyBitfield:       make([]uint8, len(att.ParticipationBitfield)),
				count:                 1,
			}
		} else {
			// update the signature
			aggregatedAttestationMap[hash].aggregateSignature.AggregateSig(sig)

			// update the participation bitfield
			for i := range aggregatedAttestationMap[hash].participationBitfield {
				aggregatedAttestationMap[hash].participationBitfield[i] |= att.ParticipationBitfield[i]
			}
			for i := range aggregatedAttestationMap[hash].custodyBitfield {
				aggregatedAttestationMap[hash].custodyBitfield[i] |= att.CustodyBitfield[i]
			}

			// keep track of how many attestations are included in the aggregated attestation
			aggregatedAttestationMap[hash].count++
		}
	}
	am.attestationsLock.Unlock()

	attestations := make([]attestationWithRealSigAndCount, 0, len(aggregatedAttestationMap))
	i := 0
	for _, att := range aggregatedAttestationMap {
		tipHash := am.blockchain.View.Chain.Tip().Hash

		tipView, err := am.blockchain.GetSubView(tipHash)
		if err != nil {
			continue
		}

		state := am.blockchain.GetState()

		tipView.SetTipSlot(state.Slot)

		err = state.ValidateAttestation(primitives.Attestation{
			AggregateSig:          att.aggregateSignature.Serialize(),
			ParticipationBitfield: att.participationBitfield,
			Data:                  att.data,
			CustodyBitfield:       att.custodyBitfield,
		}, false, &tipView, c)
		if err != nil {
			fmt.Println(err)
			continue
		}

		attestations = append(attestations, *att)
		i++
	}

	sort.Sort(byCount(attestations))

	attestationsToInclude := make([]primitives.Attestation, len(attestations))
	for i, att := range attestations {
		attestationsToInclude[i] = primitives.Attestation{
			AggregateSig:          att.aggregateSignature.Serialize(),
			ParticipationBitfield: att.participationBitfield,
			Data:                  att.data,
			CustodyBitfield:       att.custodyBitfield,
		}
	}

	return attestationsToInclude, nil
}

// removeOldAttestations removes attestations that are more than one epoch old.
func (am *attestationMempool) removeOldAttestations(slot uint64, c *config.Config) {
	am.attestationsLock.Lock()
	defer am.attestationsLock.Unlock()
	newAttestations := make([]primitives.Attestation, 0)
	for _, a := range am.attestations {
		if a.Data.Slot+c.EpochLength < slot {
			continue
		}

		newAttestations = append(newAttestations, a)
	}

	am.attestations = newAttestations
}

// RemoveAttestationsFromBitfield removes attestations that have already been included.
func (m *Mempool) RemoveAttestationsFromBitfield(slot uint64, shard uint64, bitfield []uint8) {
	am := m.AttestationMempool
	am.attestationsLock.Lock()
	defer am.attestationsLock.Unlock()
	newAttestations := make([]primitives.Attestation, 0)
	numRemoved := 0
	for _, att := range am.attestations {
		if att.Data.Slot == slot && att.Data.Shard == shard {
			intersect := false
			for i := range att.ParticipationBitfield {
				if att.ParticipationBitfield[i]&bitfield[i] != 0 {
					intersect = true
				}
			}
			if intersect {
				numRemoved++
				continue
			}
		}
		newAttestations = append(newAttestations, att)
	}

	am.attestations = newAttestations
}

// Size gets the size of the mempool.
func (am *attestationMempool) Size() int {
	am.attestationsLock.Lock()
	defer am.attestationsLock.Unlock()
	return len(am.attestations)
}
