package validator

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/go-ssz"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
)

func (v *Validator) proposeShardblock(ctx context.Context, shardID uint64, slot uint64) error {
	finalizedInfo, err := v.blockchainRPC.GetValidatorProof(v.ctx, &pb.GetValidatorProofRequest{ValidatorID: v.id})
	if err != nil {
		return err
	}

	finalizedProof, err := primitives.ValidatorProofFromProto(finalizedInfo.Proof)
	if err != nil {
		return err
	}

	template, err := v.shardRPC.GenerateBlockTemplate(ctx, &pb.BlockGenerationRequest{
		Shard:               shardID,
		Slot:                slot,
		FinalizedBeaconHash: finalizedInfo.FinalizedHash,
	})
	if err != nil {
		return err
	}

	blockTemplate, err := primitives.ShardBlockFromProto(template)
	if err != nil {
		return err
	}

	blockTemplate.Header.Validator = v.id

	blockTemplate.Header.Signature = bls.EmptySignature.Serialize()

	blockTemplate.Header.ValidatorProof = *finalizedProof

	blockHashWithoutSignature, err := ssz.HashTreeRoot(blockTemplate)
	if err != nil {
		return err
	}

	key := v.keystore.GetKeyForValidator(v.id)

	signature, err := bls.Sign(key, blockHashWithoutSignature[:], bls.DomainShardProposal)
	if err != nil {
		return err
	}

	sigBytes := signature.Serialize()

	blockTemplate.Header.Signature = sigBytes

	_, err = v.shardRPC.SubmitBlock(ctx, &pb.ShardBlockSubmission{
		Block: blockTemplate.ToProto(),
		Shard: shardID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (v *Validator) proposeBlock(ctx context.Context, information proposerAssignment) error {
	var slotBytes [8]byte
	binary.BigEndian.PutUint64(slotBytes[:], information.slot)
	slotBytesHash := chainhash.HashH(slotBytes[:])

	key := v.keystore.GetKeyForValidator(v.id)

	randaoSig, err := bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		return err
	}

	parentRootBytes, err := v.blockchainRPC.GetBlockHash(context.Background(), &pb.GetBlockHashRequest{
		SlotNumber: information.slot,
	})
	if err != nil {
		return err
	}

	parentRoot, err := chainhash.NewHash(parentRootBytes.Hash)
	if err != nil {
		return err
	}

	stateRootBytes, err := v.blockchainRPC.GetStateRoot(context.Background(), &pb.GetStateRootRequest{
		BlockHash: parentRoot[:],
	})
	if err != nil {
		return err
	}

	stateRoot, err := chainhash.NewHash(stateRootBytes.StateRoot)
	if err != nil {
		return err
	}

	mempool, err := v.blockchainRPC.GetMempool(context.Background(), &pb.MempoolRequest{
		LastBlockHash: parentRootBytes.Hash,
		Slot:          information.slot,
	})
	if err != nil {
		return err
	}

	v.logger.WithFields(logrus.Fields{
		"mempoolSize": len(mempool.Attestations) + len(mempool.Deposits) + len(mempool.CasperSlashings) + len(mempool.ProposerSlashings),
		"slot":        information.slot,
	}).Debug("creating block")

	blockBody, err := primitives.BlockBodyFromProto(mempool)
	if err != nil {
		return err
	}

	attestationsByTargetEpoch := make(map[uint64][]primitives.Attestation)

	for _, attestation := range blockBody.Attestations {
		targetEpoch := attestation.Data.TargetEpoch

		attestationsByTargetEpoch[targetEpoch] = append(attestationsByTargetEpoch[targetEpoch], attestation)
	}

	for epoch, attestations := range attestationsByTargetEpoch {
		v.logger.Debugf("Including epoch %.03d with %.02d attestations", epoch, len(attestations))
	}

	newBlock := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     information.slot,
			ParentRoot:     *parentRoot,
			StateRoot:      *stateRoot,
			RandaoReveal:   randaoSig.Serialize(),
			Signature:      bls.EmptySignature.Serialize(),
			ValidatorIndex: v.id,
		},
		BlockBody: *blockBody,
	}

	blockHash, err := ssz.HashTreeRoot(newBlock)
	if err != nil {
		return err
	}

	v.logger.Info("signing block")

	psd := primitives.ProposalSignedData{
		Slot:      information.slot,
		Shard:     config.MainNetConfig.BeaconShardNumber,
		BlockHash: blockHash,
	}

	psdHash, err := ssz.HashTreeRoot(psd)
	if err != nil {
		return err
	}

	sig, err := bls.Sign(v.keystore.GetKeyForValidator(v.id), psdHash[:], bls.DomainProposal)
	if err != nil {
		return err
	}
	newBlock.BlockHeader.Signature = sig.Serialize()
	hashWithSignature, err := ssz.HashTreeRoot(newBlock)
	if err != nil {
		return err
	}

	v.logger.WithFields(logrus.Fields{
		"blockHash": fmt.Sprintf("%x", hashWithSignature),
		"slot":      information.slot,
	}).Debug("submitting block")

	submitBlockRequest := &pb.SubmitBlockRequest{
		Block: newBlock.ToProto(),
	}

	_, err = v.blockchainRPC.SubmitBlock(context.Background(), submitBlockRequest)
	if err != nil {
		logrus.WithField("slot", information.slot).Error(err)
		return nil
	}

	return err
}
