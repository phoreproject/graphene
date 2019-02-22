package p2p

import (
	"context"
	"net"
	"time"

	proto "github.com/golang/protobuf/proto"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ps "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/beacon"
	pb "github.com/phoreproject/synapse/pb"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type messageHandler func(peer *PeerNode, message proto.Message)
type onPeerConnectedHandler func(peer *PeerNode)

// HostNode is the node for host
type HostNode struct {
	publicKey         crypto.PubKey
	privateKey        crypto.PrivKey
	host              host.Host
	ctx               context.Context
	cancel            context.CancelFunc
	grpcServer        *grpc.Server
	peerList          []PeerNode
	messageHandlerMap map[string]messageHandler
	onPeerConnected   onPeerConnectedHandler
}

var protocolID = protocol.ID("/grpc/0.0.1")

// NewHostNode creates a host node
func NewHostNode(listenAddress multiaddr.Multiaddr, publicKey crypto.PubKey, privateKey crypto.PrivKey, chain *beacon.Blockchain) (*HostNode, error) {
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

	grpcServer := grpc.NewServer()
	hostNode := HostNode{
		publicKey:         publicKey,
		privateKey:        privateKey,
		host:              host,
		ctx:               ctx,
		cancel:            cancel,
		grpcServer:        grpcServer,
		messageHandlerMap: make(map[string]messageHandler),
	}

	host.SetStreamHandler(protocolID, hostNode.handleStream)

	pb.RegisterMainRPCServer(grpcServer, NewMainRPCServer(chain))

	return &hostNode, nil
}

// handleStream handles an incoming stream.
func (node *HostNode) handleStream(stream inet.Stream) {
	node.createPeerNodeFromStream(stream)
}

func (node *HostNode) handleMessage(peer *PeerNode, message proto.Message) {
	handler, ok := node.messageHandlerMap[proto.MessageName(message)]
	if ok {
		handler(peer, message)
	}
}

// RegisterMessageHandler registers a message handler
func (node *HostNode) RegisterMessageHandler(messageName string, handler messageHandler) {
	node.messageHandlerMap[messageName] = handler
}

// SetOnPeerConnectedHandler sets the onPeerConnected handler
func (node *HostNode) SetOnPeerConnectedHandler(handler onPeerConnectedHandler) {
	node.onPeerConnected = handler
}

// Connect connects to a peer
func (node *HostNode) Connect(peerInfo *peerstore.PeerInfo) (*PeerNode, error) {
	for _, p := range node.GetHost().Peerstore().PeersWithAddrs() {
		if p == peerInfo.ID {
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

	return node.createPeerNodeFromStream(stream), nil
}

// Run runs the main loop of the host node
func (node *HostNode) createPeerNodeFromStream(stream inet.Stream) *PeerNode {
	peerNode := NewPeerNode(stream)

	node.peerList = append(node.peerList, peerNode)

	go processMessages(stream, func(message proto.Message) {
		node.handleMessage(&peerNode, message)
	})

	if node.onPeerConnected != nil {
		node.onPeerConnected(&peerNode)
	}

	return &peerNode
}

// Run runs the main loop of the host node
func (node *HostNode) Run() {
}

// GetGRPCServer returns the grpc server.
func (node *HostNode) GetGRPCServer() *grpc.Server {
	return node.grpcServer
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

// GetPeerList returns the peer list
func (node *HostNode) GetPeerList() []PeerNode {
	return node.peerList
}

// GetConnectedPeerCount returns the connected peer count
func (node *HostNode) GetConnectedPeerCount() int {
	return node.host.Peerstore().Peers().Len()
}
