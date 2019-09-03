package validator

import (
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/go-ssz"
)

func getAttestation(information attestationAssignment, blockHash chainhash.Hash) (*primitives.AttestationData, [32]byte, error) {
	a := primitives.AttestationData{
		Slot:                information.slot,
		BeaconBlockHash:     information.beaconBlockHash,
		SourceEpoch:         information.sourceEpoch,
		SourceHash:          information.sourceHash,
		TargetHash:          information.targetHash,
		TargetEpoch:         information.targetEpoch,
		Shard:               information.shard,
		ShardBlockHash:      blockHash,
		LatestCrosslinkHash: information.latestCrosslinks[information.shard].ShardBlockHash,
	}

	attestationAndPoCBit := primitives.AttestationDataAndCustodyBit{Data: a, PoCBit: false}
	hashAttestation, err := ssz.HashTreeRoot(attestationAndPoCBit)
	if err != nil {
		return nil, [32]byte{}, err
	}

	return &a, hashAttestation, nil
}

func (v *Validator) signAttestation(hashAttestation [32]byte, data primitives.AttestationData, committeeSize uint64, committeeIndex uint64) (*primitives.Attestation, error) {
	signature, err := bls.Sign(v.keystore.GetKeyForValidator(v.id), hashAttestation[:], primitives.GetDomain(*v.forkData, data.Slot, bls.DomainAttestation))
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

func (v *Validator) attestBlock(information attestationAssignment) (*primitives.Attestation, error) {
	lastEpochSlot := information.slot - (information.slot % v.config.EpochLength)

	var shardBlockHash *chainhash.Hash

	if lastEpochSlot >= v.config.EpochLength {
		shardBlockSlot := lastEpochSlot - v.config.EpochLength

		shardBlockHashResponse, err := v.shardRPC.GetBlockHashAtSlot(v.ctx, &pb.SlotRequest{
			Shard: information.shard,
			Slot:  shardBlockSlot,
		})

		if err != nil {
			return nil, err
		}

		shardBlockHash, err = chainhash.NewHash(shardBlockHashResponse.BlockHash)
		if err != nil {
			return nil, err
		}
	} else {
		shardBlock := primitives.GetGenesisBlockForShard(information.shard)
		genesisHash, err := ssz.HashTreeRoot(shardBlock)
		if err != nil {
			return nil, err
		}

		shardBlockHash = (*chainhash.Hash)(&genesisHash)
	}

	// create attestation
	attData, hash, err := getAttestation(information, *shardBlockHash)
	if err != nil {
		return nil, err
	}

	// sign attestation
	att, err := v.signAttestation(hash, *attData, information.committeeSize, information.committeeIndex)
	if err != nil {
		return nil, err
	}

	return att, nil
}
