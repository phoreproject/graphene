package p2p

import (
	iaddr "github.com/ipfs/go-ipfs-addr"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
)

// PeerInfoToBytes converts a peer info to bytes array.
func PeerInfoToBytes(peerInfo *peerstore.PeerInfo) ([]byte, error) {
	return peerInfo.MarshalJSON()
}

// BytesToPeerInfo converts a byte array to a peer info.
func BytesToPeerInfo(b []byte) (*peerstore.PeerInfo, error) {
	peerInfo := &peerstore.PeerInfo{}
	err := peerInfo.UnmarshalJSON(b)
	if err != nil {
		return nil, err
	}
	return peerInfo, nil
}

// IDToString converts ID to string
func IDToString(id peer.ID) string {
	return id.String()
}

// StringToID converts string to ID
func StringToID(s string) (peer.ID, error) {
	return peer.IDFromString(s)
}

// PeerInfoToAddrString converts a peer address to string.
func PeerInfoToAddrString(peerInfo *peerstore.PeerInfo) (string, error) {
	return peerInfo.Addrs[0].String(), nil
}

// AddrStringToPeerInfo converts a string to a peer info.
func AddrStringToPeerInfo(addrStr string) (*peerstore.PeerInfo, error) {
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
