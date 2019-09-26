package p2p

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
)

// ParseInitialConnections parses comma separated multiaddresses into peer info.
func ParseInitialConnections(in []string) ([]peer.AddrInfo, error) {
	peers := make([]peer.AddrInfo, 0)

	for _, currentAddr := range in {
		if len(currentAddr) == 0 {
			continue
		}
		maddr, err := multiaddr.NewMultiaddr(currentAddr)
		if err != nil {
			logger.WithField("addr", currentAddr).Warn("invalid multiaddr")
			continue
		}
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			logger.WithField("addr", currentAddr).Warn("invalid multiaddr")
			continue
		}

		peers = append(peers, *info)
	}

	return peers, nil
}
