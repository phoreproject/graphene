package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/phoreproject/synapse/pb"
)

var zeroHash = [32]byte{}

func (v *Validator) attestBlock(information slotInformation) error {
	attData, hash, err := v.getAttestation(information)
	if err != nil {
		return err
	}

	att, err := v.signAttestation(hash, *attData)
	if err != nil {
		return err
	}

	attProto := att.ToProto()

	attBytes, err := proto.Marshal(attProto)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("signedAttestation %d", v.slot)

	topicRequest := fmt.Sprintf("signedAttestation request %d %d", v.slot, v.shard)

	sub, err := v.p2pRPC.Subscribe(context.Background(), &pb.SubscriptionRequest{
		Topic: topicRequest,
	})

	if err != nil {
		return err
	}

	msgListener, err := v.p2pRPC.ListenForMessages(context.Background(), sub)
	if err != nil {
		return err
	}

	defer func() {
		v.p2pRPC.Unsubscribe(context.Background(), sub)
	}()

	messages := make(chan pb.Message)
	errors := make(chan error)

	go func() {
		for {
			msg, err := msgListener.Recv()
			if err != nil {
				errors <- err
				return
			}
			messages <- *msg
		}
	}()

	timeLimit := time.NewTimer(time.Second * 15)

	for {
		select {
		case msg := <-messages:
			var attRequest pb.AttestationRequest
			err = proto.Unmarshal(msg.Data, &attRequest)
			if err != nil {
				return err
			}

			if attRequest.ParticipationBitfield[v.committeeID/8]&(1<<(v.committeeID%8)) == 0 {
				_, err = v.p2pRPC.Broadcast(context.Background(), &pb.MessageAndTopic{
					Topic: topic,
					Data:  attBytes,
				})
			} else {
				msgListener.CloseSend()
				return nil
			}
		case err := <-errors:
			return err
		case <-timeLimit.C:
			msgListener.CloseSend()
			return nil
		}
	}
}
