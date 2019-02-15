package p2p

import (
	"bufio"

	proto "github.com/golang/protobuf/proto"
	crypto "github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/phoreproject/synapse/pb"
)

// PeerNode is the node for the peer
type PeerNode struct {
	publicKey crypto.PubKey
	stream    inet.Stream
	client    pb.MainRPCClient
	writer    *bufio.Writer
}

// GetClient returns the rpc client
func (node *PeerNode) GetClient() pb.MainRPCClient {
	return node.client
}

// NewPeerNode creates a P2pPeerNode
func NewPeerNode(stream inet.Stream) PeerNode {
	return PeerNode{
		stream: stream,
		writer: bufio.NewWriter(stream),
	}
}

// SendMessage sends a protobuf message to this peer
func (node *PeerNode) SendMessage(message proto.Message) {
	writeMessage(message, node.writer)
}
