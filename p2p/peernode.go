package p2p

import (
	"bufio"
	"context"
	"errors"
	"io"
	"math/rand"
	"time"

	inet "github.com/libp2p/go-libp2p-net"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	logger "github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/pb"

	"github.com/golang/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
)

// ClientVersion is the version of the client.
const ClientVersion = 0

// Peer is a representation of an external peer.
type Peer struct {
	peerInfo          *peerstore.PeerInfo
	host              *HostNode
	timeoutInterval   time.Duration
	heartbeatInterval time.Duration

	ID         peer.ID
	Outbound   bool
	Connecting bool
	// The last nonce we sent them
	LastPingNonce     uint64
	LastMessageTime   time.Time
	Version           uint64
	ProcessingRequest bool

	messageHandlers map[string]MessageHandler
	ctx             context.Context
	cancel          context.CancelFunc

	outgoingMessages chan proto.Message
}

// newPeer creates a P2pPeerNode
func newPeer(outbound bool, id peer.ID, host *HostNode, timeoutInterval time.Duration, connection inet.Stream, heartbeatInterval time.Duration) *Peer {
	ctx, cancel := context.WithCancel(context.Background())

	peer := &Peer{
		ID:              id,
		host:            host,
		timeoutInterval: timeoutInterval,

		Outbound:          outbound,
		LastPingNonce:     0,
		LastMessageTime:   time.Unix(0, 0),
		Connecting:        true,
		ProcessingRequest: false,

		messageHandlers: make(map[string]MessageHandler),
		ctx:             ctx,
		cancel:          cancel,

		heartbeatInterval: heartbeatInterval,
		outgoingMessages:  make(chan proto.Message),
	}

	go peer.handleConnection(connection)

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

func (node *Peer) sendMessages(writer *bufio.Writer) {
	for {
		select {
		case msg := <-node.outgoingMessages:
			logger.WithFields(logger.Fields{
				"peer":    node.ID,
				"message": proto.MessageName(msg),
			}).Debug("sending message")
			writeMessage(msg, writer)
		case <-node.ctx.Done():
			break
		}
	}
}

func (node *Peer) processMessages(reader *bufio.Reader) {
	err := processMessages(reader, func(message proto.Message) error {
		go func() {
			logger.WithFields(logger.Fields{
				"peer":    node.ID,
				"message": proto.MessageName(message),
			}).Debug("received message")

			err := node.handleMessage(message)
			if err != nil {
				if err != io.EOF {
					logger.Errorf("error processing message from peer %s: %s", node.ID, err)
				}
				node.cancel()
			}
		}()

		return nil
	})
	if err != nil {
		if err != io.EOF {
			logger.Errorf("error processing message from peer: %s", node.ID)
		}
	}
}

func (node *Peer) sendHeartbeat() {
	nonce := rand.Uint64()
	node.LastPingNonce = nonce
	node.SendMessage(&pb.PingMessage{
		Nonce: rand.Uint64(),
	})
}

func (node *Peer) sendHeartbeats() {
	node.sendHeartbeat()
	heartbeatTicker := time.NewTicker(node.heartbeatInterval)
	for {
		select {
		case <-heartbeatTicker.C:
			node.sendHeartbeat()
		case <-node.ctx.Done():
			break
		}
	}
}

func (node *Peer) handleConnection(connection inet.Stream) {
	go node.processMessages(bufio.NewReader(connection))

	go node.sendMessages(bufio.NewWriter(connection))

	go node.sendHeartbeats()

	logger.WithField("id", node.ID).Info("connected to peer")

	// once we're done, clean up the streams
	<-node.ctx.Done()

	connection.Reset()

	node.host.removePeer(node)
}

// SendMessage sends a protobuf message to this peer
func (node *Peer) SendMessage(message proto.Message) {
	node.outgoingMessages <- message
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
func (node *Peer) Disconnect() {
	node.cancel()
}

// Reject sends reject message and disconnect from the peer
func (node *Peer) Reject(message string) error {
	node.SendMessage(&pb.RejectMessage{
		Message: message,
	})

	node.Disconnect()
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
	node.SendMessage(&pb.VerackMessage{
		Version: ClientVersion,
		PeerID:  ourIDBytes,
	})

	return nil
}

func (node *Peer) handleVerackMessage(message *pb.VerackMessage) error {
	node.Connecting = false
	return nil
}

func (node *Peer) handlePingMessage(message *pb.PingMessage) error {
	if node.Connecting {
		return errors.New("sent ping before connecting")
	}
	node.SendMessage(&pb.PongMessage{
		Nonce: message.Nonce,
	})
	return nil
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
	node.SendMessage(&addrMessage)

	return nil
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
