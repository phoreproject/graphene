package p2p

import (
	"bufio"

	crypto "github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/rpc"
	"google.golang.org/grpc"
)

// PeerNodeHandler implements HostNodeAbstract
type PeerNodeHandler struct {
}

// P2pPeerNode is the node for the peer
type P2pPeerNode struct {
	publicKey  crypto.PubKey
	stream     inet.Stream
	readWriter *bufio.ReadWriter
	client     pb.MainRPCClient
}

// GetClient returns the rpc client
func (node *P2pPeerNode) GetClient() pb.MainRPCClient {
	return node.client
}

// Send implements PeerNode
func (node P2pPeerNode) Send(data []byte) {
	node.readWriter.Write(data)
	node.readWriter.Flush()
}

// CreatePeerNode implements HostNodeAbstract
func (handler PeerNodeHandler) CreatePeerNode(stream inet.Stream, grpcConn *grpc.ClientConn) rpc.RpcPeerNode {
	client := pb.NewMainRPCClient(grpcConn)
	return P2pPeerNode{
		stream:     stream,
		readWriter: bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream)),
		client:     client,
	}
}
