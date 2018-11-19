package net

import (
	"context"
	"io"

	"github.com/phoreproject/synapse/pb"
	"github.com/sirupsen/logrus"
)

// RPCClient is a P2P module RPC client.
type RPCClient struct {
	conn pb.P2PRPCClient
}

// NewRPCClient creates a new client for interacting with the
// P2P RPC.
func NewRPCClient(client pb.P2PRPCClient) *RPCClient {
	return &RPCClient{
		conn: client,
	}
}

// Subscribe subscribes to a topic with a handler.
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
			logrus.WithField("error", err).Warn("unhandled error while receiving messages from P2P RPC")
			return 0, err
		}

		err = handler(msg.Data)
		if err != nil {
			logrus.WithField("error", err).Warn("unhandled error while running handler from P2P RPC")
		}
	}

	return sub.ID, nil
}

// CancelSubscription cancels a subscription given the ID.
func (r *RPCClient) CancelSubscription(subID uint64) error {
	_, err := r.conn.Unsubscribe(context.Background(), &pb.Subscription{ID: subID})
	return err
}
