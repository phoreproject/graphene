package p2p

import (
	"bufio"

	crypto "github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
)

// PeerNode is the node for the peer
type PeerNode struct {
	publicKey  crypto.PubKey
	stream     inet.Stream
	readWriter *bufio.ReadWriter
}
