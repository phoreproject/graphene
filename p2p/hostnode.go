package p2p

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	ps "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type messageHandler func(peer *PeerNode, message proto.Message)
type anyMessageHandler func(peer *PeerNode, message proto.Message) bool
type onPeerConnectedHandler func(peer *PeerNode)

// HostNode is the node for p2p host
// It's the low level P2P communication layer, the App class handles high level protocols
// The RPC communication is hanlded by App, not HostNode
type HostNode struct {
	publicKey  crypto.PubKey
	privateKey crypto.PrivKey
	host       host.Host
	gossipSub  *pubsub.PubSub
	ctx        context.Context
	cancel     context.CancelFunc
	// a messageHandler is called when a message with certain name is received
	messageHandlerMap map[string]messageHandler
	// anyMessagehandler is called upon any message is received
	anyMessagehandler anyMessageHandler
	onPeerConnected   onPeerConnectedHandler

	// All peers that connected successfully with correct handshake
	livePeerList []*PeerNode
	// All live peers that are inbound
	inboundPeerList []*PeerNode
	// All live peers that are outbound
	outboundPeerList []*PeerNode
	// Connecting but not handshaked peers
	connectingPeerList []*PeerNode
}

var protocolID = protocol.ID("/grpc/phore/0.0.1")

// NewHostNode creates a host node
func NewHostNode(listenAddress multiaddr.Multiaddr, publicKey crypto.PubKey, privateKey crypto.PrivKey) (*HostNode, error) {
	ctx, cancel := context.WithCancel(context.Background())
	h, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(listenAddress),
		libp2p.Identity(privateKey),
	)
	if err != nil {
		logger.WithField("Function", "NewHostNode").Warn(err)
		cancel()
		return nil, err
	}

	for _, a := range h.Addrs() {
		logger.WithField("address", a.String()).Debug("binding to port")
	}

	if err != nil {
		cancel()
		return nil, err
	}

	g, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		cancel()
		return nil, err
	}

	hostNode := HostNode{
		publicKey:         publicKey,
		privateKey:        privateKey,
		host:              h,
		gossipSub:         g,
		ctx:               ctx,
		cancel:            cancel,
		messageHandlerMap: make(map[string]messageHandler),
	}

	h.SetStreamHandler(protocolID, hostNode.handleStream)

	return &hostNode, nil
}

// handleStream handles an incoming stream.
func (node *HostNode) handleStream(stream inet.Stream) {
	node.createPeerNodeFromStream(stream, false)
}

func (node *HostNode) handleMessage(peer *PeerNode, message proto.Message) {
	logger.Infof("Received message: %s", proto.MessageName(message))

	if node.anyMessagehandler != nil {
		if !node.anyMessagehandler(peer, message) {
			return
		}
	}

	handler, ok := node.messageHandlerMap[proto.MessageName(message)]
	if ok {
		handler(peer, message)
	}
}

// RegisterMessageHandler registers a message handler
func (node *HostNode) RegisterMessageHandler(messageName string, handler messageHandler) {
	node.messageHandlerMap[messageName] = handler
}

// SetAnyMessageHandler sets a message handler for any message
// The handler return false to prevent the message from dispatching
func (node *HostNode) SetAnyMessageHandler(handler anyMessageHandler) {
	node.anyMessagehandler = handler
}

// SetOnPeerConnectedHandler sets the onPeerConnected handler
func (node *HostNode) SetOnPeerConnectedHandler(handler onPeerConnectedHandler) {
	node.onPeerConnected = handler
}

// Connect connects to a peer
func (node *HostNode) Connect(peerInfo *peerstore.PeerInfo) (*PeerNode, error) {
	if peerInfo.ID == node.GetHost().ID() {
		return nil, errors.New("cannot connect to self")
	}

	for _, p := range node.GetHost().Peerstore().PeersWithAddrs() {
		if p == peerInfo.ID {
			logger.WithField("addrs", peerInfo.Addrs).WithField("id", peerInfo.ID).Debug("connecting to self, abort")
			return nil, nil
		}
	}

	logger.WithField("addrs", peerInfo.Addrs).WithField("id", peerInfo.ID).Debug("attempting to connect to a peer")

	node.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, ps.PermanentAddrTTL)

	stream, err := node.host.NewStream(context.Background(), peerInfo.ID, protocolID)

	if err != nil {
		logger.WithField("Function", "Connect").WithField("error", err).Warn("failed to open stream")
		return nil, err
	}

	return node.createPeerNodeFromStream(stream, true), nil
}

// Run runs the main loop of the host node
func (node *HostNode) createPeerNodeFromStream(stream inet.Stream, outbound bool) *PeerNode {
	peerNode := newPeerNode(stream, outbound)

	node.connectingPeerList = append(node.connectingPeerList, peerNode)

	go processMessages(stream, func(message proto.Message) {
		node.handleMessage(peerNode, message)
	})

	if node.onPeerConnected != nil {
		node.onPeerConnected(peerNode)
	}

	return peerNode
}

// GetDialOption returns the WithDialer option to dial via libp2p.
// note: ctx should be the root context.
func (node *HostNode) GetDialOption(ctx context.Context) grpc.DialOption {
	return grpc.WithDialer(func(peerIdStr string, timeout time.Duration) (net.Conn, error) {
		subCtx, subCtxCancel := context.WithTimeout(ctx, timeout)
		defer subCtxCancel()

		id, err := peer.IDB58Decode(peerIdStr)
		if err != nil {
			logger.WithField("Function", "GetDialOption").WithField("PeerID", peerIdStr).Warn(err)
			return nil, err
		}

		err = node.host.Connect(subCtx, ps.PeerInfo{
			ID: id,
		})
		if err != nil {
			logger.WithField("Function", "GetDialOption").Warn(err)
			return nil, err
		}

		stream, err := node.host.NewStream(ctx, id, protocolID)
		if err != nil {
			logger.WithField("Function", "GetDialOption").Warn(err)
			return nil, err
		}

		return &streamConn{Stream: stream}, nil
	})
}

// Dial attempts to open a GRPC connection over libp2p to a peer.
// Note that the context is used as the **stream context** not just the dial context.
func (node *HostNode) Dial(ctx context.Context, peerID peer.ID, dialOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialOpsPrepended := append([]grpc.DialOption{node.GetDialOption(ctx)}, dialOpts...)
	return grpc.DialContext(ctx, peerID.Pretty(), dialOpsPrepended...)
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

// Broadcast broadcasts a message to the network for a topic.
func (node *HostNode) Broadcast(topic string, data []byte) error {
	return node.gossipSub.Publish(topic, data)
}

// SubscribeMessage registers a handler for a network topic.
func (node *HostNode) SubscribeMessage(topic string, handler func(*PeerNode, []byte)) (*pubsub.Subscription, error) {
	subscription, err := node.gossipSub.Subscribe(topic)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			msg, err := subscription.Next(node.ctx)
			if err != nil {
				logger.WithField("error", err).Warn("error when getting next topic message")
				continue
			}

			id, err := peer.IDFromBytes(msg.From)
			if err != nil {
				logger.WithField("error", err).Warn("error when getting ID from topic message")
				continue
			}

			p := node.FindPeerByID(id)
			if p != nil {
				logger.Warn("Can't find p from ID")
				continue
			}

			handler(p, msg.Data)
		}
	}()

	return subscription, nil
}

// UnsubscribeMessage cancels a subscription to a topic.
func (node *HostNode) UnsubscribeMessage(subscription *pubsub.Subscription) {
	subscription.Cancel()
}

// GetLivePeerList returns all peer list, including both inbound and outbound
func (node *HostNode) GetLivePeerList() []*PeerNode {
	return node.livePeerList
}

// GetInboundPeerList returns the inbound peer list
func (node *HostNode) GetInboundPeerList() []*PeerNode {
	return node.inboundPeerList
}

// GetOutboundPeerList returns the outbound peer list
func (node *HostNode) GetOutboundPeerList() []*PeerNode {
	return node.outboundPeerList
}

// PeerDoneHandShake is called when handshaking is finished
func (node *HostNode) PeerDoneHandShake(peer *PeerNode) {
	for i, p := range node.connectingPeerList {
		if p == peer {
			node.connectingPeerList = append(node.connectingPeerList[:i], node.connectingPeerList[i+1:]...)
			break
		}
	}
	node.livePeerList = append(node.livePeerList, peer)
	if peer.IsOutbound() {
		node.outboundPeerList = append(node.outboundPeerList, peer)
	} else {
		node.inboundPeerList = append(node.inboundPeerList, peer)
	}
}

func (node *HostNode) removePeer(peer *PeerNode) {
	for i, p := range node.livePeerList {
		if p == peer {
			node.livePeerList = append(node.livePeerList[:i], node.livePeerList[i+1:]...)
			break
		}
	}
	for i, p := range node.outboundPeerList {
		if p == peer {
			node.outboundPeerList = append(node.outboundPeerList[:i], node.outboundPeerList[i+1:]...)
			break
		}
	}
	for i, p := range node.inboundPeerList {
		if p == peer {
			node.inboundPeerList = append(node.inboundPeerList[:i], node.inboundPeerList[i+1:]...)
			break
		}
	}
}

// DisconnectPeer disconnects a peer
func (node *HostNode) DisconnectPeer(peer *PeerNode) {
	peer.disconnect()
	node.removePeer(peer)
}

// FindPeerByID finds a peer node by ID, returns nil if not found
func (node *HostNode) FindPeerByID(id peer.ID) *PeerNode {
	for _, p := range node.livePeerList {
		if p.GetID() == id {
			return p
		}
	}
	return nil
}
