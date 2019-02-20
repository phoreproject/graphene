package p2p

import (
	"context"
	"net"

	logger "github.com/sirupsen/logrus"

	inet "github.com/libp2p/go-libp2p-net"
	mnet "github.com/multiformats/go-multiaddr-net"
)

// streamConn represents a net.Conn wrapped to be compatible with net.conn
type streamConn struct {
	inet.Stream
}

// LocalAddr returns the local address.
func (c *streamConn) LocalAddr() net.Addr {
	addr, err := mnet.ToNetAddr(c.Stream.Conn().LocalMultiaddr())
	if err != nil {
		return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	}
	return addr
}

// RemoteAddr returns the remote address.
func (c *streamConn) RemoteAddr() net.Addr {
	addr, err := mnet.ToNetAddr(c.Stream.Conn().RemoteMultiaddr())
	if err != nil {
		return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	}
	return addr
}

// grpcListener implements the net.Listener interface.
type grpcListener struct {
	*HostNode
	listenerCtx       context.Context
	listenerCtxCancel context.CancelFunc
}

// newGrpcListener creates a new GRPC listener.
func newGrpcListener(hostNode *HostNode) net.Listener {
	listener := &grpcListener{
		HostNode: hostNode,
	}
	listener.listenerCtx, listener.listenerCtxCancel = context.WithCancel(hostNode.ctx)
	return listener
}

// Accept implements net.Listener.
func (listener *grpcListener) Accept() (net.Conn, error) {
	logger.Debug("grpcListener: Accepted new connection")
	/*
		select {
		case <-listener.listenerCtx.Done():
			return nil, io.EOF
		case stream := <-listener.streamCh:
			return &streamConn{Stream: stream}, nil
		}
	*/
	return nil, nil
}

// Addr implements net.Listener.
func (listener *grpcListener) Addr() net.Addr {
	listenAddrs := listener.host.Network().ListenAddresses()
	if len(listenAddrs) > 0 {
		for _, addr := range listenAddrs {
			if na, err := mnet.ToNetAddr(addr); err == nil {
				return na
			}
		}
	}
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
}

// Close implements net.Listener.
func (listener *grpcListener) Close() error {
	listener.listenerCtxCancel()
	return nil
}
