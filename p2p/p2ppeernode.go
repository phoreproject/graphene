package p2p

import (
	"bufio"

	proto "github.com/golang/protobuf/proto"
	inet "github.com/libp2p/go-libp2p-net"
)

// P2pPeerNode is the base interface of a peer node
type P2pPeerNode struct {
	stream inet.Stream
	writer *bufio.Writer
}

// NewP2pPeerNode creates a P2pPeerNode
func NewP2pPeerNode(stream inet.Stream) P2pPeerNode {
	return P2pPeerNode{
		stream: stream,
		writer: bufio.NewWriter(stream),
	}
}

// SendMessage sends a protobuf message
func (node *P2pPeerNode) SendMessage(message proto.Message) {
	writeMessage(message, node.writer)
}
