package net

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-crypto"

	"github.com/libp2p/go-libp2p-peerstore"

	logger "github.com/sirupsen/logrus"

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
				logger.WithField("error", err).Warn("error when getting next topic message")
				return
			}

			err = handler(msg.Data)
			if err != nil {
				logger.WithField("topic", topic).WithField("error", err).Warn("error when handling message")
			}
		}
	}()

	return s, nil
}

// Broadcast broadcasts a message to the network for a topic.
func (n *NetworkingService) Broadcast(topic string, data []byte) error {
	return n.gossipSub.Publish(topic, data)
}

// CancelHandler cancels a subscription to a topic.
func (n *NetworkingService) CancelHandler(subscription *pubsub.Subscription) {
	subscription.Cancel()
}

// GetPeers gets the multiaddrs for all of the peers.
func (n *NetworkingService) GetPeers() []multiaddr.Multiaddr {
	peers := n.host.Peerstore().PeersWithAddrs()
	addrs := []multiaddr.Multiaddr{}
	for _, p := range peers {
		peerAddrs := n.host.Peerstore().Addrs(p)
		for _, addr := range peerAddrs {
			addrs = append(addrs, addr)
		}
	}
	return addrs
}

// IsConnected checks if the networking service is connected
// to any peers.
func (n *NetworkingService) IsConnected() bool {
	return n.host.Peerstore().Peers().Len() > 0
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
		logger.WithField("address", a.String()).Debug("binding to port")
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

	err = StartDiscovery(ctx, host, NewDiscoveryOptions())
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
