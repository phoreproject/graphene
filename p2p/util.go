package p2p

import (
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"strings"

	logger "github.com/sirupsen/logrus"
)

// ParseInitialConnections parses comma separated multiaddresses into peer info.
func ParseInitialConnections(in string) ([]peerstore.PeerInfo, error) {
	logrus.SetLevel(logrus.DebugLevel)

	peerStrings := strings.Split(in, ",")
	peers := make([]peerstore.PeerInfo, 0)

	for _, currentAddr := range peerStrings {
		if len(currentAddr) == 0 {
			continue
		}
		maddr, err := multiaddr.NewMultiaddr(currentAddr)
		if err != nil {
			logger.WithField("addr", currentAddr).Warn("invalid multiaddr")
			continue
		}
		info, err := peerstore.InfoFromP2pAddr(maddr)
		if err != nil {
			logger.WithField("addr", currentAddr).Warn("invalid multiaddr")
			continue
		}

		peers = append(peers, *info)
	}

	return peers, nil
}
