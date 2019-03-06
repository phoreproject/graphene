package p2p

import (
	iaddr "github.com/ipfs/go-ipfs-addr"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
)

// StringToPeerInfo converts a string to a peer info.
func StringToPeerInfo(addrStr string) (*peerstore.PeerInfo, error) {
	addr, err := iaddr.ParseString(addrStr)
	if err != nil {
		return nil, err
	}
	peerinfo, err := peerstore.InfoFromP2pAddr(addr.Multiaddr())
	if err != nil {
		return nil, err
	}
	return peerinfo, nil
}
