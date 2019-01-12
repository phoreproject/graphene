package p2p

import (
	crypto "github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/phoreproject/synapse/net"
	"github.com/phoreproject/synapse/pb"
	"google.golang.org/grpc"
)

// PeerNodeHandler implements HostNodeAbstract
type PeerNodeHandler struct {
}

// P2pPeerNode is the node for the peer
type P2pPeerNode struct {
	publicKey crypto.PubKey
	stream    inet.Stream
	client    pb.MainRPCClient
}

// GetClient returns the rpc client
func (node *P2pPeerNode) GetClient() pb.MainRPCClient {
	return node.client
}

// createPeerNode implements HostNodeAbstract
func (handler PeerNodeHandler) CreatePeerNode(stream inet.Stream, grpcConn *grpc.ClientConn) net.PeerNode {
	client := pb.NewMainRPCClient(grpcConn)
	return P2pPeerNode{
		stream: stream,
		client: client,
	}
}
