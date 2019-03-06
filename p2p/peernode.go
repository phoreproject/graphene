package p2p

import (
	"bufio"

	"github.com/phoreproject/synapse/pb"

	proto "github.com/golang/protobuf/proto"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
)

// PeerNode is the node for the peer
type PeerNode struct {
	stream   inet.Stream
	writer   *bufio.Writer
	outbound bool
	id       peer.ID

	LastPingNonce   uint64
	LastPingTime    uint64
	LastMessageTime uint64
}

// newPeerNode creates a P2pPeerNode
func newPeerNode(stream inet.Stream, outbound bool) *PeerNode {
	return &PeerNode{
		stream:   stream,
		writer:   bufio.NewWriter(stream),
		outbound: outbound,

		LastPingNonce:   0,
		LastPingTime:    0,
		LastMessageTime: 0,
	}
}

// SendMessage sends a protobuf message to this peer
func (node *PeerNode) SendMessage(message proto.Message) {
	writeMessage(message, node.writer)
}

// IsOutbound returns true if the connection is an outbound
func (node *PeerNode) IsOutbound() bool {
	return node.outbound
}

// IsInbound returns true if the connection is an inbound
func (node *PeerNode) IsInbound() bool {
	return !node.IsOutbound()
}

// GetID returns the ID
func (node *PeerNode) GetID() peer.ID {
	return node.id
}

func (node *PeerNode) disconnect() {
	node.stream.Reset()
}

// HandleVersionMessage handles VersionMessage
func (node *PeerNode) HandleVersionMessage(message *pb.VersionMessage) {
	node.id, _ = peer.IDFromBytes(message.Id)
}
