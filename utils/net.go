package utils

import (
	"fmt"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"net"
)

// NetAddrToGRPCDialString converts a net address to a GRPC dial string.
func NetAddrToGRPCDialString(addr net.Addr) string {
	if addr.Network() == "unix" {
		return fmt.Sprintf("%s://%s", addr.Network(), addr.String())
	}
	return addr.String()
}

// MultiaddrStringToDialString converts an address string to a dial string.
func MultiaddrStringToDialString(addrString string) (string, error) {
	ma, err := multiaddr.NewMultiaddr(addrString)
	if err != nil {
		return "", err
	}

	addr, err := manet.ToNetAddr(ma)
	if err != nil {
		return "", err
	}

	return NetAddrToGRPCDialString(addr), nil
}
