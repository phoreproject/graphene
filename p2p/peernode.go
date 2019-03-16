package p2p

import (
	"bufio"
	"errors"
	"github.com/libp2p/go-libp2p-peerstore"
	"time"

	"github.com/phoreproject/synapse/pb"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-peer"
)

// ClientVersion is the version of the client.
const ClientVersion = 0

// Peer is a representation of an external peer.
type Peer struct {
	stream   *bufio.ReadWriter
	peerInfo *peerstore.PeerInfo
	host     *HostNode

	ID              peer.ID
	Outbound        bool
	Connecting      bool
	LastPingNonce   uint64
	LastPingTime    uint64
	LastMessageTime uint64
	Version         uint64
}

// newPeer creates a P2pPeerNode
func newPeer(stream *bufio.ReadWriter, outbound bool, id peer.ID, host *HostNode) *Peer {
	return &Peer{
		stream: stream,
		ID:     id,
		host:   host,

		Outbound:        outbound,
		LastPingNonce:   0,
		LastPingTime:    0,
		LastMessageTime: 0,
		Connecting:      true,
	}
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

func (node *Peer) disconnect() error {
	return nil
}

// Reject sends reject message and disconnect from the peer
func (node *Peer) Reject(message string) error {
	err := node.SendMessage(&pb.RejectMessage{
		Message: message,
	})
	if err != nil {
		return err
	}

	err = node.disconnect()
	if err != nil {
		return err
	}

	return nil
}

// HandleVersionMessage handles VersionMessage from this peer
func (node *Peer) HandleVersionMessage(message *pb.VersionMessage) error {
	peerID, err := peer.IDFromBytes(message.PeerID)
	if err != nil {
		return err
	}
	node.ID = peerID
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

func (node *Peer) handleMessage(message proto.Message) error {
	node.LastMessageTime = uint64(time.Now().Unix())
	return nil
}
