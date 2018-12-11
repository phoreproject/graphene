package p2p

import (
	"context"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	multiaddr "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
)

// PeerNode is the node for the peer
type PeerNode struct {
	publicKey crypto.PubKey
}

// HostNode is the node for host
type HostNode struct {
	publicKey  crypto.PubKey
	privateKey crypto.PrivKey
	host       host.Host
	gossipSub  *pubsub.PubSub
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewHostNode creates a host node
func (node *HostNode) NewHostNode(listenAddress *multiaddr.Multiaddr, publicKey crypto.PubKey, privateKey crypto.PrivKey) (*HostNode, error) {
	ctx, cancel := context.WithCancel(context.Background())
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(*listenAddress),
		libp2p.Identity(privateKey),
	)

	for _, a := range host.Addrs() {
		logger.WithField("address", a.String()).Debug("binding to port")
	}

	if err != nil {
		cancel()
		return nil, err
	}

	g, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		cancel()
		return nil, err
	}

	hostNode := HostNode{
		publicKey:  publicKey,
		privateKey: privateKey,
		host:       host,
		gossipSub:  g,
		ctx:        ctx,
		cancel:     cancel,
	}

	return &hostNode, nil
}

// GetPublicKey returns the public key
func (node *HostNode) GetPublicKey() *crypto.PubKey {
	return &node.publicKey
}

// GetContext returns the context
func (node *HostNode) GetContext() context.Context {
	return node.ctx
}

// GetHost returns the host
func (node *HostNode) GetHost() host.Host {
	return node.host
}

// Discover discovers the peers
func (node *HostNode) Discover(options *DiscoveryOptions) error {
	return startDiscovery(node, options)
}

// GetConnectedPeerCount returns the connected peer count
func (node *HostNode) GetConnectedPeerCount() int {
	return node.host.Peerstore().Peers().Len()
}

// Broadcast broadcasts a message to the network for a topic.
func (node *HostNode) Broadcast(topic string, data []byte) error {
	return node.gossipSub.Publish(topic, data)
}

// Connect connects to a peer
// TODO: should return PeerNode
func (node *HostNode) Connect(p *peerstore.PeerInfo) error {
	ctx, cancel := context.WithTimeout(node.ctx, 10*time.Second)
	defer cancel()
	return node.host.Connect(ctx, *p)
}
