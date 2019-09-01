package validator

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/prysmaticlabs/go-ssz"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
)

func (v *Validator) proposeShardblock(ctx context.Context, shardID uint64, slot uint64, finalizedHash chainhash.Hash) error {
	template, err := v.shardRPC.GenerateBlockTemplate(ctx, &pb.BlockGenerationRequest{
		Shard:               shardID,
		Slot:                slot,
		FinalizedBeaconHash: finalizedHash[:], // TODO: fix this later (this is actually called using justified hash)
	})
	if err != nil {
		return err
	}

	blockTemplate, err := primitives.ShardBlockFromProto(template)
	if err != nil {
		return err
	}

	blockTemplate.Header.Signature = bls.EmptySignature.Serialize()

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
	stateRootBytes, err := v.blockchainRPC.GetStateRoot(context.Background(), &empty.Empty{})
	if err != nil {
		return err
	}

	stateRoot, err := chainhash.NewHash(stateRootBytes.StateRoot)
	if err != nil {
		return err
	}

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

	mempool, err := v.blockchainRPC.GetMempool(context.Background(), &pb.MempoolRequest{
		LastBlockHash: parentRootBytes.Hash,
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

	newBlock := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   information.slot,
			ParentRoot:   *parentRoot,
			StateRoot:    *stateRoot,
			RandaoReveal: randaoSig.Serialize(),
			Signature:    bls.EmptySignature.Serialize(),
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
