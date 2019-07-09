package beacon

import (
	"errors"
	"sort"
	"sync"

	"github.com/phoreproject/synapse/pb"
	ssz "github.com/prysmaticlabs/go-ssz"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
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
		attHash, err := ssz.HashTreeRoot(primitives.AttestationDataAndCustodyBit{Data: a.Data, PoCBit: false})
		if err != nil {
			continue
		}
		m.RemoveAttestationsFromBitfield(attHash, a.ParticipationBitfield)
	}
}

type attestationMempool struct {
	blockchain       *Blockchain
	attestations     map[chainhash.Hash][]primitives.Attestation // maps the hashed data
	attestationsLock *sync.RWMutex
}

func newAttestationMempool(blockchain *Blockchain) *attestationMempool {
	return &attestationMempool{
		attestations:     make(map[chainhash.Hash][]primitives.Attestation),
		attestationsLock: new(sync.RWMutex),
		blockchain:       blockchain,
	}
}

// ProcessNewAttestation processes a new attestation to be included in a block.
func (m *Mempool) ProcessNewAttestation(att primitives.Attestation) error {
	m.AttestationMempool.attestationsLock.Lock()
	defer m.AttestationMempool.attestationsLock.Unlock()

	attHash, err := ssz.HashTreeRoot(primitives.AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
	if err != nil {
		return err
	}

	if atts, found := m.AttestationMempool.attestations[attHash]; found {
		for _, a := range atts {
			for i, b := range a.ParticipationBitfield {
				if b&att.ParticipationBitfield[i] != 0 {
					logrus.Debug("duplicate attestation, ignoring")
					return nil
				}
			}
		}

		m.AttestationMempool.attestations[attHash] = append(m.AttestationMempool.attestations[attHash], att)
	} else {
		m.AttestationMempool.attestations[attHash] = []primitives.Attestation{att}
	}
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
func (m *Mempool) GetAttestationsToInclude(slot uint64, lastBlockHash chainhash.Hash, c *config.Config) ([]primitives.Attestation, error) {
	// include any attestations
	aggregatedAttestationMap := make(map[chainhash.Hash]*attestationWithRealSigAndCount)

	am := m.AttestationMempool

	am.removeOldAttestations(slot, c)

	am.attestationsLock.RLock()

	state, found := m.blockchain.stateManager.GetStateForHash(lastBlockHash)
	if !found {
		return nil, errors.New("don't have state for block hash")
	}

	blockView, err := m.blockchain.GetSubView(lastBlockHash)
	if err != nil {
		return nil, err
	}

	stateCopy := state.Copy()

	_, err = stateCopy.ProcessSlots(slot+1, &blockView, c)
	if err != nil {
		return nil, err
	}

	// go through all of the (separate) attestations we've received
	for hash, atts := range am.attestations {
		for _, att := range atts {
			if att.Data.Slot+c.MinAttestationInclusionDelay > slot {
				continue // don't include attestations that aren't valid yet
			}

			// too soon
			if att.Data.TargetEpoch > state.EpochIndex {
				continue
			}

			if att.Data.Slot+c.EpochLength <= slot {
				continue
			}

			if aggAtt, found := aggregatedAttestationMap[hash]; !found {
				err := stateCopy.ValidateAttestation(primitives.Attestation{
					AggregateSig:          att.AggregateSig,
					ParticipationBitfield: att.ParticipationBitfield,
					Data:                  att.Data,
					CustodyBitfield:       att.CustodyBitfield,
				}, true, c)
				if err != nil {
					continue
				}

				// get the signature of the attestation
				sig, err := bls.DeserializeSignature(att.AggregateSig)
				if err != nil {
					return nil, err
				}

				// if this isn't already included, include it
				aggregatedAttestationMap[hash] = &attestationWithRealSigAndCount{
					data:                  att.Data.Copy(),
					aggregateSignature:    sig,
					participationBitfield: make([]uint8, len(att.ParticipationBitfield)),
					custodyBitfield:       make([]uint8, len(att.CustodyBitfield)),
					count:                 1,
				}

				copy(aggregatedAttestationMap[hash].participationBitfield, att.ParticipationBitfield)
				copy(aggregatedAttestationMap[hash].custodyBitfield, att.CustodyBitfield)
			} else {
				intersects := false

				for i, b := range aggAtt.participationBitfield {
					if b&att.ParticipationBitfield[i] != 0 {
						intersects = true
					}
				}

				if intersects {
					continue
				}

				sig, err := bls.DeserializeSignature(att.AggregateSig)
				if err != nil {
					return nil, err
				}

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
	}
	am.attestationsLock.RUnlock()

	attestations := make([]attestationWithRealSigAndCount, 0, len(aggregatedAttestationMap))
	i := 0
	for _, att := range aggregatedAttestationMap {
		attestations = append(attestations, *att)
		i++
	}

	sort.Sort(byCount(attestations))

	numAttestationsToInclude := len(attestations)
	if numAttestationsToInclude > c.MaxAttestations {
		numAttestationsToInclude = c.MaxAttestations
	}

	attestationsToInclude := make([]primitives.Attestation, numAttestationsToInclude)
	for i := 0; i < numAttestationsToInclude; i++ {
		attestationsToInclude[i] = primitives.Attestation{
			AggregateSig:          attestations[i].aggregateSignature.Serialize(),
			ParticipationBitfield: attestations[i].participationBitfield,
			Data:                  attestations[i].data,
			CustodyBitfield:       attestations[i].custodyBitfield,
		}
	}

	return attestationsToInclude, nil
}

// removeOldAttestations removes attestations that are more than one epoch old.
func (am *attestationMempool) removeOldAttestations(slot uint64, c *config.Config) {
	am.attestationsLock.Lock()
	defer am.attestationsLock.Unlock()
	for h, atts := range am.attestations {
		if len(atts) == 0 {
			delete(am.attestations, h)
		}

		a := atts[0]

		if a.Data.Slot+c.EpochLength <= slot {
			delete(am.attestations, h)
		}
	}
}

// RemoveAttestationsFromBitfield removes attestations that have already been included.
func (m *Mempool) RemoveAttestationsFromBitfield(h chainhash.Hash, bitfield []uint8) {
	am := m.AttestationMempool
	am.attestationsLock.Lock()
	defer am.attestationsLock.Unlock()

	if atts, found := am.attestations[h]; found {
		newAttestations := make([]primitives.Attestation, 0, len(atts))

		for _, att := range atts {
			oldBitfield := att.ParticipationBitfield
			intersect := false
			for i := range oldBitfield {
				if oldBitfield[i]&bitfield[i] != 0 {
					intersect = true
				}
			}
			if !intersect {
				newAttestations = append(newAttestations, att)
			}
		}

		if len(newAttestations) == 0 {
			delete(am.attestations, h)
		} else {
			am.attestations[h] = newAttestations
		}
	}
}

// Size gets the size of the mempool.
func (am *attestationMempool) Size() int {
	am.attestationsLock.RLock()
	defer am.attestationsLock.RUnlock()
	return len(am.attestations)
}

func doAttestationsIntersect(participationBitfield []byte, att *primitives.Attestation) (bool, error) {
	if len(participationBitfield) != len(att.ParticipationBitfield) {
		return false, errors.New("participation bitfield did not match")
	}

	for i := range att.ParticipationBitfield {
		if att.ParticipationBitfield[i]&participationBitfield[i] != 0 {
			return true, nil
		}
	}

	return false, nil
}

// GetMempoolDifference gets the difference between our mempool and a peer's
func (am *attestationMempool) GetMempoolDifference(message *pb.GetMempoolMessage) ([]primitives.Attestation, error) {
	difference := make([]primitives.Attestation, 0)

	am.attestationsLock.RLock()
	defer am.attestationsLock.RUnlock()

	attestationsInMessage := make(map[chainhash.Hash]struct{})

	for _, mempoolInv := range message.Attestations {
		attestationHash, err := chainhash.NewHash(mempoolInv.AttestationHash)
		if err != nil {
			return nil, err
		}

		attestationsInMessage[*attestationHash] = struct{}{}

		atts, found := am.attestations[*attestationHash]
		if !found {
			// we don't have the attestation
			continue
		} else {
			// we do have the attestation, so figure out which attestations to send
			for _, att := range atts {
				intersect, err := doAttestationsIntersect(att.ParticipationBitfield, &att)
				if err != nil {
					return nil, err
				}

				if !intersect {
					difference = append(difference, att)
				}
			}
		}
	}

	for h, atts := range am.attestations {
		if _, found := attestationsInMessage[h]; !found {
			difference = append(difference, atts...)
		}
	}

	return difference, nil
}

// GetMempoolSummary gets a summary of the mempool so we can sync it with other peers.
func (am *attestationMempool) GetMempoolSummary() *pb.GetMempoolMessage {
	am.attestationsLock.RLock()
	defer am.attestationsLock.RUnlock()

	summary := &pb.GetMempoolMessage{
		Attestations: make([]*pb.AttestationMempoolItem, len(am.attestations)),
	}

	i := 0
	for h, atts := range am.attestations {
		if len(atts) == 0 {
			continue
		}

		participationBitfield := atts[0].ParticipationBitfield[:]

		for _, p := range atts[1:] {
			for i := range participationBitfield {
				participationBitfield[i] |= p.ParticipationBitfield[i]
			}
		}

		summary.Attestations[i] = &pb.AttestationMempoolItem{
			AttestationHash: h[:],
			Participation:   participationBitfield,
		}
		i++
	}

	return summary
}
