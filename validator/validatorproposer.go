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

	attestations, err := v.mempool.attestationMempool.getAttestationsToInclude(information.slot, v.config)
	if err != nil {
		return err
	}

	v.logger.WithFields(logrus.Fields{
		"attestationSize": len(attestations),
		"mempoolSize":     v.mempool.attestationMempool.size(),
		"slot":            information.slot,
	}).Debug("creating block")

	stateRootBytes, err := v.blockchainRPC.GetStateRoot(context.Background(), &empty.Empty{})
	if err != nil {
		return err
	}

	stateRoot, err := chainhash.NewHash(stateRootBytes.StateRoot)
	if err != nil {
		return err
	}

	slots, err := v.blockchainRPC.GetProposerSlots(context.Background(), &pb.GetProposerSlotsRequest{ValidatorID: v.id})
	if err != nil {
		return err
	}

	var proposerSlotsBytes [8]byte
	binary.BigEndian.PutUint64(proposerSlotsBytes[:], slots.ProposerSlots)

	key := v.keystore.GetKeyForValidator(v.id)

	randaoSig, err := bls.Sign(key, proposerSlotsBytes[:], bls.DomainRandao)
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

	newBlock := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   information.slot,
			ParentRoot:   *parentRoot,
			StateRoot:    *stateRoot,
			RandaoReveal: randaoSig.Serialize(),
			Signature:    bls.EmptySignature.Serialize(),
		},
		BlockBody: primitives.BlockBody{
			Attestations:      attestations,
			ProposerSlashings: []primitives.ProposerSlashing{},
			CasperSlashings:   []primitives.CasperSlashing{},
			Deposits:          []primitives.Deposit{},
			Exits:             []primitives.Exit{},
		},
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

	v.logger.WithFields(logrus.Fields{
		"blockHash": fmt.Sprintf("%x", hashWithSignature),
		"slot":      information.slot,
	}).Debug("submitted block")

	for _, a := range attestations {
		v.mempool.attestationMempool.removeAttestationsFromBitfield(a.Data.Slot, a.Data.Shard, a.ParticipationBitfield)
	}

	return err
}
