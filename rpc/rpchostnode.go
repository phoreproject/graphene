package rpc

import (
	"context"
	"net"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ps "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	multiaddr "github.com/multiformats/go-multiaddr"
	pb "github.com/phoreproject/synapse/pb"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// RpcHostNodeAbstract simulates abstract class in other modern languages such as Java/C++
type RpcHostNodeAbstract interface {
	CreatePeerNode(stream inet.Stream, grpcConn *grpc.ClientConn) RpcPeerNode
}

// RpcHostNode is the node for host
type RpcHostNode struct {
	publicKey  crypto.PubKey
	privateKey crypto.PrivKey
	host       host.Host
	gossipSub  *pubsub.PubSub
	ctx        context.Context
	cancel     context.CancelFunc
	grpcServer *grpc.Server
	streamCh   chan inet.Stream
	peerList   []RpcPeerNode
	abstract   RpcHostNodeAbstract
}

var protocolID = protocol.ID("/grpc/0.0.1")

// NewHostNode creates a host node
func NewHostNode(
	listenAddress multiaddr.Multiaddr,
	publicKey crypto.PubKey,
	privateKey crypto.PrivKey,
	server pb.MainRPCServer,
	abstract RpcHostNodeAbstract) (*RpcHostNode, error) {
	ctx, cancel := context.WithCancel(context.Background())
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(listenAddress),
		libp2p.Identity(privateKey),
	)
	if err != nil {
		logger.WithField("Function", "NewHostNode").Warn(err)
		cancel()
		return nil, err
	}

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

	grpcServer := grpc.NewServer()
	stream := make(chan inet.Stream)
	hostNode := RpcHostNode{
		publicKey:  publicKey,
		privateKey: privateKey,
		host:       host,
		gossipSub:  g,
		ctx:        ctx,
		cancel:     cancel,
		grpcServer: grpcServer,
		streamCh:   stream,
		abstract:   abstract,
	}

	host.SetStreamHandler(protocolID, hostNode.handleStream)

	pb.RegisterMainRPCServer(grpcServer, server)

	return &hostNode, nil
}

// handleStream handles an incoming stream.
func (node *RpcHostNode) handleStream(stream inet.Stream) {
	select {
	case <-node.ctx.Done():
		return
	case node.streamCh <- stream:
	}
}

// Connect connects to a peer
func (node *RpcHostNode) Connect(peerInfo *peerstore.PeerInfo) (RpcPeerNode, error) {
	for _, p := range node.GetHost().Peerstore().PeersWithAddrs() {
		if p == peerInfo.ID {
			return nil, nil
		}
	}

	logger.WithField("addrs", peerInfo.Addrs).WithField("id", peerInfo.ID).Debug("attempting to connect to a peer")

	ctx, cancel := context.WithTimeout(node.ctx, 10*time.Second)
	defer cancel()

	node.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, ps.PermanentAddrTTL)

	grpcConn, err := node.Dial(ctx, peerInfo.ID, grpc.WithInsecure(), grpc.WithBlock())

	if err != nil {
		logger.WithField("Function", "Connect").WithField("error", err).Warn("failed to connect to peer")
		return nil, err
	}

	stream, err := node.host.NewStream(context.Background(), peerInfo.ID, protocolID)

	if err != nil {
		logger.WithField("Function", "Connect").WithField("error", err).Warn("failed to open stream")
		return nil, err
	}

	peerNode := node.abstract.CreatePeerNode(stream, grpcConn)

	node.peerList = append(node.peerList, peerNode)

	return peerNode, err
}

// Run runs the main loop of the host node
func (node *RpcHostNode) Run() {
	go node.grpcServer.Serve(newGrpcListener(node))
}

// GetGRPCServer returns the grpc server.
func (node *RpcHostNode) GetGRPCServer() *grpc.Server {
	return node.grpcServer
}

// GetDialOption returns the WithDialer option to dial via libp2p.
// note: ctx should be the root context.
func (node *RpcHostNode) GetDialOption(ctx context.Context) grpc.DialOption {
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
func (node *RpcHostNode) Dial(ctx context.Context, peerID peer.ID, dialOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialOpsPrepended := append([]grpc.DialOption{node.GetDialOption(ctx)}, dialOpts...)
	return grpc.DialContext(ctx, peerID.Pretty(), dialOpsPrepended...)
}

// GetPublicKey returns the public key
func (node *RpcHostNode) GetPublicKey() *crypto.PubKey {
	return &node.publicKey
}

// GetContext returns the context
func (node *RpcHostNode) GetContext() context.Context {
	return node.ctx
}

// GetHost returns the host
func (node *RpcHostNode) GetHost() host.Host {
	return node.host
}

// GetPeerList returns the peer list
func (node *RpcHostNode) GetPeerList() []RpcPeerNode {
	return node.peerList
}

// GetConnectedPeerCount returns the connected peer count
func (node *RpcHostNode) GetConnectedPeerCount() int {
	return node.host.Peerstore().Peers().Len()
}

// Broadcast broadcasts a message to the network for a topic.
func (node *RpcHostNode) Broadcast(topic string, data []byte) error {
	return node.gossipSub.Publish(topic, data)
}

// SubscribeMessage registers a handler for a network topic.
func (node *RpcHostNode) SubscribeMessage(topic string, handler func([]byte) error) (*pubsub.Subscription, error) {
	subscription, err := node.gossipSub.Subscribe(topic)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			msg, err := subscription.Next(node.ctx)
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

	return subscription, nil
}

// UnsubscribeMessage cancels a subscription to a topic.
func (node *RpcHostNode) UnsubscribeMessage(subscription *pubsub.Subscription) {
	subscription.Cancel()
}
