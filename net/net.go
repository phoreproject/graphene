package net

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
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
		logger.Debug("got new block", "data", string(b.Data))

		// newBlock := primitives.Block{}

		// buf := bytes.NewBuffer(b.Data)

		// binary.Read(buf, binary.BigEndian, &newBlock)

		// n.blocks <- newBlock
	}
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

// NewNetworkingService creates a networking service instance that will
// run on the given IP.
func NewNetworkingService(addr *multiaddr.Multiaddr, privateKey crypto.PrivKey) (NetworkingService, error) {
	ctx := context.Background()
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(*addr),
		libp2p.Identity(privateKey),
	)

	for _, a := range host.Addrs() {
		logger.Debug("binding to port", "address", a.String())
	}

	if err != nil {
		return NetworkingService{}, err
	}

	g, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return NetworkingService{}, err
	}

	err = startDiscovery(ctx, host)
	if err != nil {
		return NetworkingService{}, err
	}

	n := NetworkingService{
		host:      host,
		blocks:    make(chan primitives.Block),
		gossipSub: g,
	}

	timer := time.NewTimer(time.Second * 2)

	go func() {
		<-timer.C
		i0, _ := rand.Int(rand.Reader, big.NewInt(100000))
		i := i0.Uint64()

		toPublish := []byte(fmt.Sprintf("test data %d", i))

		g.Publish("block", toPublish)

		logger.Debug("publishing", "data", string(toPublish))
	}()

	s, err := g.Subscribe("block")
	if err != nil {
		return n, err
	}

	go n.handleBlockSubscriptions(s)

	return n, nil
}
