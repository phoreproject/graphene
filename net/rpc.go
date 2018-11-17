package net

import (
	"context"
	"io"

	"github.com/inconshreveable/log15"
	"github.com/phoreproject/synapse/pb"
)

type RPCClient struct {
	conn pb.P2PRPCClient
}

func NewRPCClient(client pb.P2PRPCClient) *RPCClient {
	return &RPCClient{
		conn: client,
	}
}

func (r *RPCClient) Subscribe(topic string, handler func([]byte) error) (uint64, error) {
	sub, err := r.conn.Subscribe(context.Background(), &pb.SubscriptionRequest{Topic: topic})
	if err != nil {
		return 0, err
	}

	listener, err := r.conn.ListenForMessages(context.Background(), sub)
	if err != nil {
		return 0, err
	}

	for {
		msg, err := listener.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log15.Warn("unhandled error while receiving messages from P2P RPC", "error", err)
			return 0, err
		}

		err = handler(msg.Data)
		if err != nil {
			log15.Warn("unhandled error while running handler from P2P RPC", "error", err)
		}
	}

	return sub.ID, nil
}

func (r *RPCClient) CancelSubscription(subID uint64) error {
	_, err := r.conn.Unsubscribe(context.Background(), &pb.Subscription{ID: subID})
	return err
}
