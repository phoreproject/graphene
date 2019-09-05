package gotest

import (
	"crypto/rand"
	"flag"
	"strings"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"

	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/p2p"
	logger "github.com/sirupsen/logrus"
)

type goTestApp struct {
	hostNode         *p2p.HostNode
	listenAddress    ma.Multiaddr
	peerAddresses    []peerstore.PeerInfo
	discoveryOptions p2p.DiscoveryOptions
	commands         []string
}

type mockBlockChain struct {
}

func (b mockBlockChain) Height() uint64 {
	return 1
}

func (b mockBlockChain) GenesisHash() chainhash.Hash {
	bytes := make([]byte, chainhash.HashSize)
	h, err := chainhash.NewHash(bytes)
	if err != nil {
		panic(err)
	}
	return *h
}

func (app *goTestApp) run() {
	app.parseArgs()

	app.createHostNode()
}

func (app *goTestApp) parseArgs() {
	listenAddress := flag.String("listen", "/ip4/0.0.0.0/tcp/21781", "specifies the address to listen on")
	initialConnections := flag.String("connect", "", "comma separated multiaddrs")
	commands := flag.String("commands", "", "comma separated commands")

	addr, err := ma.NewMultiaddr(*listenAddress)
	if err != nil {
		panic(err)
	}
	app.listenAddress = addr

	initialPeers, err := p2p.ParseInitialConnections(*initialConnections)
	if err != nil {
		panic(err)
	}
	app.peerAddresses = initialPeers

	app.discoveryOptions.PeerAddresses = append(app.discoveryOptions.PeerAddresses, initialPeers...)

	app.discoveryOptions = p2p.NewDiscoveryOptions()

	app.commands = strings.Split(*commands, ",")
}

func (app *goTestApp) getHostKey() (crypto.PrivKey, crypto.PubKey, error) {
	k, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	if pub == nil {
		pub = k.GetPublic()
	}

	return k, pub, nil
}

func (app *goTestApp) createHostNode() {
	priv, pub, err := app.getHostKey()
	if err != nil {
		panic(err)
	}

	hostNode, err := p2p.NewHostNode(
		app.listenAddress,
		pub,
		priv,
		app.discoveryOptions,
		16*time.Second,
		8,
		8*time.Second,
		mockBlockChain{})
	if err != nil {
		panic(err)
	}
	app.hostNode = hostNode
}

func main() {
	logger.Info("Test")

	app := goTestApp{}
	app.run()
}
