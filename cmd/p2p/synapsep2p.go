package main

import (
	"flag"

	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/p2p/app"

	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
)

func parseInitialConnections(in string) ([]*peerstore.PeerInfo, error) {
	logrus.SetLevel(logrus.DebugLevel)

	currentAddr := ""

	peers := []*peerstore.PeerInfo{}

	for i := range in {
		if in[i] == ',' {
			peerinfo, err := p2p.StringToPeerInfo(currentAddr)
			if err != nil {
				return nil, err
			}
			currentAddr = ""

			peers = append(peers, peerinfo)
		}
		currentAddr = currentAddr + string(in[i])
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
	app := app.NewP2PApp(config)

	err = app.Run()
	if err != nil {
		panic(err)
	}
}
