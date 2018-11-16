package net

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-crypto"

	"github.com/libp2p/go-libp2p-peerstore"

	logger "github.com/inconshreveable/log15"

	"github.com/multiformats/go-multiaddr"

	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// NetworkingService handles networking throughout the network.
type NetworkingService struct {
	host      host.Host
	gossipSub *pubsub.PubSub
	ctx       context.Context
	cancel    context.CancelFunc
}

// Connect connects to a peer
func (n *NetworkingService) Connect(p *peerstore.PeerInfo) error {
	ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
	defer cancel()
	return n.host.Connect(ctx, *p)
}

// RegisterHandler registers a handler for a network topic.
func (n *NetworkingService) RegisterHandler(topic string, handler func([]byte) error) (*pubsub.Subscription, error) {
	s, err := n.gossipSub.Subscribe(topic)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			msg, err := s.Next(n.ctx)
			if err != nil {
				logger.Warn("error when getting next topic message", "error", err)
				return
			}

			err = handler(msg.Data)
			if err != nil {
				logger.Warn("error when handling message", "topic", topic, "error", err)
			}
		}
	}()

	return s, nil
}

func (n *NetworkingService) CancelHandler(subscription *pubsub.Subscription) {
	subscription.Cancel()
}

// NewNetworkingService creates a networking service instance that will
// run on the given IP.
func NewNetworkingService(addr *multiaddr.Multiaddr, privateKey crypto.PrivKey) (NetworkingService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(*addr),
		libp2p.Identity(privateKey),
	)

	for _, a := range host.Addrs() {
		logger.Debug("binding to port", "address", a.String())
	}

	if err != nil {
		cancel()
		return NetworkingService{}, err
	}

	g, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		cancel()
		return NetworkingService{}, err
	}

	err = startDiscovery(ctx, host)
	if err != nil {
		cancel()
		return NetworkingService{}, err
	}

	n := NetworkingService{
		host:      host,
		gossipSub: g,
		ctx:       ctx,
		cancel:    cancel,
	}

	return n, nil
}
