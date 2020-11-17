package p2p

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p"

	"github.com/libp2p/go-libp2p-peerstore/pstoremem"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

// HostNodeOptions are options passed to the host node.
type HostNodeOptions struct {
	ListenAddresses    []multiaddr.Multiaddr
	PrivateKey         crypto.PrivKey
	ConnManagerOptions ConnectionManagerOptions
	Timeout            time.Duration
}

const timeoutInterval = 60 * time.Second
const heartbeatInterval = 20 * time.Second

// HostNode is the node for p2p host
// It's the low level P2P communication layer, the App class handles high level protocols
// The RPC communication is handled by App, not HostNode
type HostNode struct {
	privateKey crypto.PrivKey

	host      host.Host
	gossipSub *pubsub.PubSub
	ctx       context.Context

	timeoutInterval   time.Duration
	heartbeatInterval time.Duration

	// discovery handles peer discovery (mDNS, DHT, etc)
	discovery *ConnectionManager
}

// NewHostNode creates a host node
func NewHostNode(ctx context.Context, options HostNodeOptions) (*HostNode, error) {
	ps := pstoremem.NewPeerstore()

	h, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(options.ListenAddresses...),
		libp2p.Identity(options.PrivateKey),
		libp2p.EnableRelay(),
		libp2p.Peerstore(ps),
	)

	if err != nil {
		return nil, err
	}

	addrs, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{
		ID:    h.ID(),
		Addrs: options.ListenAddresses,
	})
	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		logrus.WithField("addr", a).Info("binding to address")
	}

	// setup gossip sub protocol
	g, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}

	hostNode := &HostNode{
		privateKey:        options.PrivateKey,
		host:              h,
		gossipSub:         g,
		ctx:               ctx,
		timeoutInterval:   timeoutInterval,
		heartbeatInterval: heartbeatInterval,
	}

	discovery, err := NewConnectionManager(ctx, hostNode, options.ConnManagerOptions)
	if err != nil {
		return nil, err
	}
	hostNode.discovery = discovery

	h.Network().Notify(discovery)

	return hostNode, nil
}

// GetContext returns the context
func (node *HostNode) GetContext() context.Context {
	return node.ctx
}

// GetHost returns the host
func (node *HostNode) GetHost() host.Host {
	return node.host
}

// Broadcast broadcasts a message to the network for a topic.
func (node *HostNode) Broadcast(topic string, data []byte) error {
	logrus.WithField("topic", topic).WithField("size", len(data)).Debug("broadcasting data to topic")
	return node.gossipSub.Publish(topic, data)
}

// SubscribeMessage registers a handler for a network topic.
func (node *HostNode) SubscribeMessage(topic string, handler func([]byte, peer.ID)) (*pubsub.Subscription, error) {
	subscription, err := node.gossipSub.Subscribe(topic)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			msg, err := subscription.Next(node.ctx)
			logrus.WithFields(logrus.Fields{
				"peer":  msg.GetFrom(),
				"topic": topic,
				"size":  len(msg.Data),
			}).Debug("got broadcast from peer")
			if err != nil {
				logrus.WithField("error", err).Warn("error when getting next topic message")
				continue
			}

			handler(msg.Data, msg.GetFrom())
		}
	}()

	return subscription, nil
}

// UnsubscribeMessage cancels a subscription to a topic.
func (node *HostNode) UnsubscribeMessage(subscription *pubsub.Subscription) {
	subscription.Cancel()
}

func (node *HostNode) removePeer(p peer.ID) {
	node.host.Peerstore().ClearAddrs(p)
}

// DisconnectPeer disconnects a peer
func (node *HostNode) DisconnectPeer(p peer.ID) error {
	return node.host.Network().ClosePeer(p)
}

// IsConnected checks if the host node is connected.
func (node *HostNode) IsConnected() bool {
	return node.PeersConnected() > 0
}

// PeersConnected checks how many peers are connected.
func (node *HostNode) PeersConnected() int {
	return len(node.host.Network().Peers())
}

// GetPeerList returns a list of all peers.
func (node *HostNode) GetPeerList() []peer.ID {
	return node.host.Network().Peers()
}

// ConnectedToPeer returns true if we're connected to the peer.
func (node *HostNode) ConnectedToPeer(id peer.ID) bool {
	connectedness := node.host.Network().Connectedness(id)
	return connectedness == network.Connected
}

// Notify notifies a notifee for network events.
func (node *HostNode) Notify(notifee network.Notifiee) {
	node.host.Network().Notify(notifee)
}

// setStreamHandler sets a stream handler for the host node.
func (node *HostNode) setStreamHandler(id protocol.ID, handleStream func(s network.Stream)) {
	node.host.SetStreamHandler(id, handleStream)
}

// CountPeers counts the number of peers that support the protocol.
func (node *HostNode) CountPeers(id protocol.ID) int {
	count := 0
	for _, n := range node.host.Peerstore().Peers() {
		if sup, err := node.host.Peerstore().SupportsProtocols(n, string(id)); err != nil && len(sup) != 0 {
			count++
		}
	}
	return count
}

// Connect connects to peer
func (node *HostNode) Connect(ctx context.Context, pi peer.AddrInfo) error {
	logrus.WithField("peer", pi).Info("connecting to peer")
	return node.host.Connect(ctx, pi)
}

// OpenStreams opens streams to peer after connecting.
func (node *HostNode) OpenStreams(id peer.ID, protocols ...protocol.ID) error {
	protoStrings := make([]string, len(protocols))
	for i := range protocols {
		protoStrings[i] = string(protocols[i])
	}

	for _, p := range protoStrings {
		logrus.WithField("peerID", id).WithField("protocol", p).Debug("setting up stream")
		stream, err := node.host.NewStream(node.ctx, id, protocol.ID(p))
		if err != nil {
			return err
		}

		err = node.discovery.HandleOutgoing(protocol.ID(p), stream)
		if err != nil {
			return err
		}
	}

	return nil
}

// RegisterProtocolHandler registers a new protocol.
func (node *HostNode) RegisterProtocolHandler(id protocol.ID, maxPeers int, minPeers int) (*ProtocolHandler, error) {
	return node.discovery.RegisterProtocolHandler(id, maxPeers, minPeers)
}

// GetPeerDirection gets the direction of the peer.
func (node *HostNode) GetPeerDirection(id peer.ID) network.Direction {
	conns := node.host.Network().ConnsToPeer(id)

	if len(conns) != 1 {
		return network.DirUnknown
	}
	return conns[0].Stat().Direction
}
