package p2p

import (
	crypto "github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/phoreproject/synapse/pb"
)

// PeerNode is the node for the peer
type PeerNode struct {
	publicKey crypto.PubKey
	stream    inet.Stream
	client    pb.MainRPCClient
}

// GetClient returns the rpc client
func (node *PeerNode) GetClient() pb.MainRPCClient {
	return node.client
}
