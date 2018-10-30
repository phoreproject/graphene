package net

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/libp2p/go-libp2p-peerstore"

	"github.com/phoreproject/synapse/primitives"

	"github.com/multiformats/go-multiaddr"

	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// NetworkingService handles networking throughout the network.
type NetworkingService struct {
	host      host.Host
	gossipSub *pubsub.PubSub
	blocks    chan primitives.Block
}

func (n *NetworkingService) handleBlockSubscriptions(blockSub *pubsub.Subscription) error {
	defer blockSub.Cancel()
	for {
		b, err := blockSub.Next(context.Background())
		if err != nil {
			return err
		}
		newBlock := primitives.Block{}

		buf := bytes.NewBuffer(b.Data)

		binary.Read(buf, binary.BigEndian, &newBlock)

		n.blocks <- newBlock
	}
}

// GetBlocksChannel returns a channel containing incoming blocks.
func (n *NetworkingService) GetBlocksChannel() chan primitives.Block {
	return n.blocks
}

// Connect connects to a peer
func (n *NetworkingService) Connect(p *peerstore.PeerInfo) error {
	return n.host.Connect(context.Background(), *p)
}

// NewNetworkingService creates a networking service instance that will
// run on the given IP.
func NewNetworkingService(addr *multiaddr.Multiaddr) (NetworkingService, error) {
	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(*addr),
	)
	if err != nil {
		return NetworkingService{}, err
	}

	g, err := pubsub.NewGossipSub(context.Background(), host)
	if err != nil {
		return NetworkingService{}, err
	}

	n := NetworkingService{
		host:      host,
		blocks:    make(chan primitives.Block),
		gossipSub: g,
	}

	s, err := g.Subscribe("block")
	if err != nil {
		return NetworkingService{}, err
	}
	n.handleBlockSubscriptions(s)

	return n, nil
}
