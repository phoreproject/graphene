package net

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-crypto"

	"github.com/libp2p/go-libp2p-peerstore"

	logger "github.com/inconshreveable/log15"

	"github.com/phoreproject/synapse/primitives"

	"github.com/multiformats/go-multiaddr"

	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Message is a network message and the context for processing
// that message.
type Message struct {
	Ctx  context.Context
	Data []byte
}

// NetworkingService handles networking throughout the network.
type NetworkingService struct {
	host         host.Host
	gossipSub    *pubsub.PubSub
	blocks       chan primitives.Block
	ctx          context.Context
	cancel       context.CancelFunc
	closeSignals map[string]chan struct{}
}

// GetBlocksChannel returns a channel containing incoming blocks.
func (n *NetworkingService) GetBlocksChannel() chan primitives.Block {
	return n.blocks
}

// Connect connects to a peer
func (n *NetworkingService) Connect(p *peerstore.PeerInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return n.host.Connect(ctx, *p)
}

// RegisterHandler registers a handler for a network topic.
func (n *NetworkingService) RegisterHandler(topic string, handler func(Message) error) error {
	s, err := n.gossipSub.Subscribe(topic)
	if err != nil {
		return err
	}

	closeChan := make(chan struct{})

	n.closeSignals[topic] = closeChan

	go func() {
		defer s.Cancel()
		for {
			message := make(chan *pubsub.Message)
			ctx, cancel := context.WithCancel(n.ctx)
			go func() {
				m, err := s.Next(ctx)
				if err != nil {
					return
				}
				message <- m
			}()

			select {
			case m := <-message:
				msg := Message{n.ctx, m.Data}

				err = handler(msg)
				if err != nil {
					logger.Debug("message processing error", "type", topic, "error", err)
					continue
				}
			case <-closeChan:
				cancel()
				n.closeSignals[topic] = nil
				return
			}
		}
	}()

	return nil
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
		host:         host,
		blocks:       make(chan primitives.Block),
		gossipSub:    g,
		ctx:          ctx,
		cancel:       cancel,
		closeSignals: make(map[string]chan struct{}),
	}

	return n, nil
}
