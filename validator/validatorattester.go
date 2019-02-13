package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/phoreproject/synapse/pb"
)

var zeroHash = [32]byte{}

func (v *Validator) attestBlock(information assignment) error {
	// create attestation
	attData, hash, err := getAttestation(information)
	if err != nil {
		return err
	}

	// sign attestation
	att, err := v.signAttestation(hash, *attData, information.committeeSize, information.committeeIndex)
	if err != nil {
		return err
	}

	// broadcast attestation
	attProto := att.ToProto()

	attBytes, err := proto.Marshal(attProto)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("attestations epoch %d", information.slot/v.config.EpochLength)

	timeWait := time.NewTimer(time.Second * 3)

	<-timeWait.C

	_, err = v.p2pRPC.Broadcast(context.Background(), &pb.MessageAndTopic{
		Topic: topic,
		Data:  attBytes,
	})
	return err
}
