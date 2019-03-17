package main

import (
	"flag"
	"github.com/multiformats/go-multiaddr"
	"strings"

	"github.com/phoreproject/synapse/p2p/app"

	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
)

func parseInitialConnections(in string) ([]*peerstore.PeerInfo, error) {
	logrus.SetLevel(logrus.DebugLevel)

	peerStrings := strings.Split(in, ",")
	peers := make([]*peerstore.PeerInfo, 0)

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

		peers = append(peers, info)
	}

	return peers, nil
}

func main() {
	listen := flag.String("listen", "/ip4/0.0.0.0/tcp/11781", "specifies the address to listen on")
	initialConnections := flag.String("connect", "", "comma separated multiaddrs")
	rpcConnect := flag.String("rpclisten", "127.0.0.1:11783", "host and port for RPC server to listen on")
	flag.Parse()

	logger.Debug("starting p2p service")

	logger.Info("initializing net")
	ps, err := parseInitialConnections(*initialConnections)
	if err != nil {
		panic(err)
	}

	config := app.NewConfig()
	config.ListeningAddress = *listen
	config.RPCAddress = *rpcConnect
	config.AddedPeers = ps
	a := app.NewP2PApp(config)

	err = a.Initialize()
	if err != nil {
		panic(err)
	}

	err = a.Run()
	if err != nil {
		panic(err)
	}
}
