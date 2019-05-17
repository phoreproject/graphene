package p2p

import (
	"bufio"
	"errors"
	"time"

	inet "github.com/libp2p/go-libp2p-net"
	peerstore "github.com/libp2p/go-libp2p-peerstore"

	"github.com/phoreproject/synapse/pb"

	"github.com/golang/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
)

// ClientVersion is the version of the client.
const ClientVersion = 0

// Peer is a representation of an external peer.
type Peer struct {
	stream          *bufio.ReadWriter
	peerInfo        *peerstore.PeerInfo
	host            *HostNode
	timeoutInterval time.Duration

	ID                peer.ID
	Outbound          bool
	Connecting        bool
	LastPingNonce     uint64
	LastPingTime      time.Time
	LastMessageTime   time.Time
	Version           uint64
	ProcessingRequest bool

	connection      inet.Stream
	messageHandlers map[string]MessageHandler
}

// newPeer creates a P2pPeerNode
func newPeer(stream *bufio.ReadWriter, outbound bool, id peer.ID, host *HostNode, timeoutInterval time.Duration, connection inet.Stream) *Peer {
	peer := &Peer{
		stream:          stream,
		ID:              id,
		host:            host,
		timeoutInterval: timeoutInterval,

		Outbound:          outbound,
		LastPingNonce:     0,
		LastPingTime:      time.Unix(0, 0),
		LastMessageTime:   time.Unix(0, 0),
		Connecting:        true,
		ProcessingRequest: false,

		connection:      connection,
		messageHandlers: make(map[string]MessageHandler),
	}

	peer.registerMessageHandler("pb.VersionMessage", func(peer *Peer, message proto.Message) error {
		return peer.HandleVersionMessage(message.(*pb.VersionMessage))
	})

	peer.registerMessageHandler("pb.VerackMessage", func(peer *Peer, message proto.Message) error {
		return peer.handleVerackMessage(message.(*pb.VerackMessage))
	})

	peer.registerMessageHandler("pb.PingMessage", func(peer *Peer, message proto.Message) error {
		return peer.handlePingMessage(message.(*pb.PingMessage))
	})

	peer.registerMessageHandler("pb.PongMessage", func(peer *Peer, message proto.Message) error {
		return peer.handlePongMessage(message.(*pb.PongMessage))
	})

	peer.registerMessageHandler("pb.GetAddrMessage", func(peer *Peer, message proto.Message) error {
		return peer.handleGetAddrMessage(message.(*pb.GetAddrMessage))
	})

	peer.registerMessageHandler("pb.AddrMessage", func(peer *Peer, message proto.Message) error {
		return peer.handleAddrMessage(message.(*pb.AddrMessage))
	})

	return peer
}

// SendMessage sends a protobuf message to this peer
func (node *Peer) SendMessage(message proto.Message) error {
	return writeMessage(message, node.stream.Writer)
}

// IsOutbound returns true if the connection is an outbound
func (node *Peer) IsOutbound() bool {
	return node.Outbound
}

// IsInbound returns true if the connection is an inbound
func (node *Peer) IsInbound() bool {
	return !node.Outbound
}

// GetPeerInfo returns the peer info
func (node *Peer) GetPeerInfo() *peerstore.PeerInfo {
	return node.peerInfo
}

// Disconnect disconnects from a peer cleanly
func (node *Peer) Disconnect() error {
	return node.connection.Reset()
}

// Reject sends reject message and disconnect from the peer
func (node *Peer) Reject(message string) error {
	err := node.SendMessage(&pb.RejectMessage{
		Message: message,
	})
	if err != nil {
		return err
	}

	err = node.Disconnect()
	if err != nil {
		return err
	}

	return nil
}

// IsConnected checks if the peers is considered connected.
func (node *Peer) IsConnected() bool {
	return node.peerInfo != nil && time.Since(node.LastMessageTime) <= node.timeoutInterval
}

// HandleVersionMessage handles VersionMessage from this peer
func (node *Peer) HandleVersionMessage(message *pb.VersionMessage) error {
	peerID, err := peer.IDFromBytes(message.PeerID)
	if err != nil {
		return err
	}
	node.ID = peerID

	peerInfo := peerstore.PeerInfo{}
	if peerInfo.UnmarshalJSON(message.PeerInfo) == nil {
		node.peerInfo = &peerInfo
	}

	ourIDBytes, err := node.host.host.ID().MarshalBinary()
	if err != nil {
		return err
	}
	node.Connecting = false
	return node.SendMessage(&pb.VerackMessage{
		Version: ClientVersion,
		PeerID:  ourIDBytes,
	})
}

func (node *Peer) handleVerackMessage(message *pb.VerackMessage) error {
	node.Connecting = false
	return nil
}

func (node *Peer) handlePingMessage(message *pb.PingMessage) error {
	if node.Connecting {
		return errors.New("sent ping before connecting")
	}
	return node.SendMessage(&pb.PongMessage{
		Nonce: message.Nonce,
	})
}

func (node *Peer) handlePongMessage(message *pb.PongMessage) error {
	if node.LastPingNonce != message.Nonce {
		// ban peer
		return errors.New("invalid pong nonce")
	}
	return nil
}

func (node *Peer) handleGetAddrMessage(message *pb.GetAddrMessage) error {
	addrMessage := pb.AddrMessage{
		Addrs: [][]byte{},
	}
	for _, peer := range node.host.GetPeerList() {
		if peer.IsConnected() {
			data, err := peer.GetPeerInfo().MarshalJSON()
			if err == nil {
				addrMessage.Addrs = append(addrMessage.Addrs, data)
			}
		}
	}
	return node.SendMessage(&addrMessage)
}

func (node *Peer) handleAddrMessage(message *pb.AddrMessage) error {
	for _, data := range message.Addrs {
		peerInfo := peerstore.PeerInfo{}
		if peerInfo.UnmarshalJSON(data) == nil {
			if peerInfo.ID != node.host.GetHost().ID() {
				node.host.PeerDiscovered(peerInfo)
			}
		}
	}
	return nil
}

func (node *Peer) registerMessageHandler(messageName string, handler MessageHandler) {
	node.messageHandlers[messageName] = handler
}

func (node *Peer) handleMessage(message proto.Message) error {
	node.LastMessageTime = time.Now()

	name := proto.MessageName(message)

	if handler, found := node.messageHandlers[name]; found {
		handler(node, message)
	}
	return nil
}
