package validator

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

func (v *Validator) proposeBlock(ctx context.Context, information proposerAssignment) error {
	mempool, err := v.blockchainRPC.GetMempool(context.Background(), &empty.Empty{})
	if err != nil {
		return err
	}

	v.logger.WithFields(logrus.Fields{
		"mempoolSize": len(mempool.Attestations) + len(mempool.Deposits) + len(mempool.CasperSlashings) + len(mempool.ProposerSlashings),
		"slot":        information.slot,
	}).Debug("creating block")

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

	parentRootBytes, err := v.blockchainRPC.GetLastBlockHash(context.Background(), &empty.Empty{})
	if err != nil {
		return err
	}

	parentRoot, err := chainhash.NewHash(parentRootBytes.Hash)
	if err != nil {
		return err
	}

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

	blockHash, err := ssz.TreeHash(newBlock)
	if err != nil {
		return err
	}

	v.logger.Info("signing block")

	psd := primitives.ProposalSignedData{
		Slot:      information.slot,
		Shard:     config.MainNetConfig.BeaconShardNumber,
		BlockHash: blockHash,
	}

	psdHash, err := ssz.TreeHash(psd)
	if err != nil {
		return err
	}

	sig, err := bls.Sign(v.keystore.GetKeyForValidator(v.id), psdHash[:], bls.DomainProposal)
	if err != nil {
		return err
	}
	newBlock.BlockHeader.Signature = sig.Serialize()
	hashWithSignature, err := ssz.TreeHash(newBlock)
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
