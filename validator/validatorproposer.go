package validator

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
)

func (v *Validator) proposeBlock(information proposerAssignment) error {
	// wait for slot to happen to submit
	timer := time.NewTimer(time.Until(time.Unix(int64(information.proposeAt), 0)))
	<-timer.C

	for {
		canRetry, err := v.tryProposeBlock(information)
		if err != nil {
			if canRetry {
				<- time.NewTimer(time.Second * 10).C
			} else {
				return err
			}
		} else {
			return nil
		}
	}
}

func (v *Validator) tryProposeBlock(information proposerAssignment) (bool, error) {
	mempool, err := v.blockchainRPC.GetMempool(context.Background(), &empty.Empty{})
	if err != nil {
		return false, err
	}

	v.logger.WithFields(logrus.Fields{
		"mempoolSize": len(mempool.Attestations) + len(mempool.Deposits) + len(mempool.CasperSlashings) + len(mempool.ProposerSlashings),
		"slot":        information.slot,
	}).Debug("creating block")

	stateRootBytes, err := v.blockchainRPC.GetStateRoot(context.Background(), &empty.Empty{})
	if err != nil {
		return false, err
	}

	stateRoot, err := chainhash.NewHash(stateRootBytes.StateRoot)
	if err != nil {
		return false, err
	}

	var slotBytes [8]byte
	binary.BigEndian.PutUint64(slotBytes[:], information.slot)
	slotBytesHash := chainhash.HashH(slotBytes[:])

	key := v.keystore.GetKeyForValidator(v.id)

	randaoSig, err := bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		return false, err
	}

	parentRootBytes, err := v.blockchainRPC.GetLastBlockHash(context.Background(), &empty.Empty{})
	if err != nil {
		return false, err
	}

	parentRoot, err := chainhash.NewHash(parentRootBytes.Hash)
	if err != nil {
		return false, err
	}

	blockBody, err := primitives.BlockBodyFromProto(mempool)

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
		return false, err
	}

	v.logger.Info("signing block")

	psd := primitives.ProposalSignedData{
		Slot:      information.slot,
		Shard:     config.MainNetConfig.BeaconShardNumber,
		BlockHash: blockHash,
	}

	psdHash, err := ssz.TreeHash(psd)
	if err != nil {
		return false, err
	}

	sig, err := bls.Sign(v.keystore.GetKeyForValidator(v.id), psdHash[:], bls.DomainProposal)
	if err != nil {
		return false, err
	}
	newBlock.BlockHeader.Signature = sig.Serialize()
	hashWithSignature, err := ssz.TreeHash(newBlock)
	if err != nil {
		return false, err
	}

	v.logger.WithFields(logrus.Fields{
		"blockHash": fmt.Sprintf("%x", hashWithSignature),
		"slot":      information.slot,
	}).Debug("submitting block")

	submitBlockRequest := &pb.SubmitBlockRequest{
		Block: newBlock.ToProto(),
	}

	response, err := v.blockchainRPC.SubmitBlock(context.Background(), submitBlockRequest)
	if err != nil {
		logrus.WithField("slot", information.slot).Error(err)
		return false, nil
	}
	if response.CanRetry {
		return true, fmt.Errorf("retry")
	}

	v.logger.WithFields(logrus.Fields{
		"blockHash": fmt.Sprintf("%x", hashWithSignature),
		"slot":      information.slot,
	}).Debug("submitted block")

	return false, err
}
