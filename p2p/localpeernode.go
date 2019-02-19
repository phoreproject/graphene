package p2p

import (
	"bufio"

	crypto "github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/rpc"
	"google.golang.org/grpc"
)

// LocalPeerNodeHandler implements HostNodeAbstract
type LocalPeerNodeHandler struct {
}

// LocalPeerNode is the node for the peer
type LocalPeerNode struct {
	publicKey  crypto.PubKey
	stream     inet.Stream
	readWriter *bufio.ReadWriter
	client     pb.MainRPCClient
}

// GetClient returns the rpc client
func (node *LocalPeerNode) GetClient() pb.MainRPCClient {
	return node.client
}

// Send implements PeerNode
func (node LocalPeerNode) Send(data []byte) {
	node.readWriter.Write(data)
	node.readWriter.Flush()
}

// CreatePeerNode implements PeerNodeCreator
func (handler LocalPeerNodeHandler) CreatePeerNode(stream inet.Stream, grpcConn *grpc.ClientConn) rpc.RpcPeerNode {
	client := pb.NewMainRPCClient(grpcConn)
	return LocalPeerNode{
		stream:     stream,
		readWriter: bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream)),
		client:     client,
	}
}
