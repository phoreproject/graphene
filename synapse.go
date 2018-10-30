package main

import (
	"flag"
	"fmt"

	"github.com/whyrusleeping/go-logging"

	"github.com/libp2p/go-libp2p-peerstore"

	"github.com/multiformats/go-multiaddr"

	"github.com/phoreproject/synapse/blockchain"
	"github.com/phoreproject/synapse/db"
	"github.com/phoreproject/synapse/net"
)

var log = logging.MustGetLogger("main")

func parseInitialConnections(in string) (peerstore.PeerInfo, error) {
	currentAddr := ""

	peers := peerstore.PeerInfo{}

	for i := range in {
		if in[i] == ',' {
			sourceMultiAddr, err := multiaddr.NewMultiaddr(currentAddr)
			if err != nil {
				return peers, err
			}
			peers.Addrs = append(peers.Addrs, sourceMultiAddr)
		}
		currentAddr = currentAddr + string(in[i])
	}

	return peers, nil
}

const clientVersion = "0.0.1"

func main() {
	log.Infof("Starting client version %s", clientVersion)
	database := db.NewInMemoryDB()
	c := blockchain.MainNetConfig
	blockchain := blockchain.NewBlockchain(database, &c)

	listen := flag.String("listen", "/ip4/0.0.0.0/tcp/11781", "specifies the address to listen on")
	initialConnections := flag.String("connect", "", "comma separated multiaddrs")
	flag.Parse()

	ps, err := parseInitialConnections(*initialConnections)
	if err != nil {
		panic(err)
	}

	sourceMultiAddr, err := multiaddr.NewMultiaddr(*listen)
	if err != nil {
		fmt.Printf("address %s is invalid", *listen)
		return
	}

	network, err := net.NewNetworkingService(&sourceMultiAddr)
	if err != nil {
		panic(err)
	}

	network.Connect(&ps)

	blocks := network.GetBlocksChannel()

	go blockchain.HandleNewBlocks(blocks)
}
