package main

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"flag"
	mrand "math/rand"
	"strings"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
	logger "github.com/sirupsen/logrus"
)

func makeRandomBytes(length int) []byte {
	result := make([]byte, length)
	mrand.Read(result)
	return result
}

type goTestApp struct {
	hostNode         *p2p.HostNode
	listenAddress    ma.Multiaddr
	discoveryOptions p2p.DiscoveryOptions
	commands         []string
	genesisHash      chainhash.Hash
}

func (app *goTestApp) Height() uint64 {
	return 1
}

func (app *goTestApp) GenesisHash() chainhash.Hash {
	return app.genesisHash
}

func (app *goTestApp) run() {
	app.parseArgs()

	app.createHostNode()

	time.Sleep(10 * time.Second)

	app.executeCommands()

	time.Sleep(5 * time.Second)

	app.disconnectAllPeers()
}

func (app *goTestApp) parseArgs() {
	listenAddress := flag.String("listen", "/ip4/0.0.0.0/tcp/21781", "specifies the address to listen on")
	initialConnections := flag.String("connect", "", "comma separated multiaddrs")
	commands := flag.String("commands", "", "comma separated commands")
	genesis := flag.String("genesis", "", "Genesis hash")

	flag.Parse()

	addr, err := ma.NewMultiaddr(*listenAddress)
	if err != nil {
		panic(err)
	}
	app.listenAddress = addr

	initialPeers, err := p2p.ParseInitialConnections(*initialConnections)
	if err != nil {
		panic(err)
	}

	app.discoveryOptions = p2p.NewDiscoveryOptions()

	app.discoveryOptions.PeerAddresses = append(app.discoveryOptions.PeerAddresses, initialPeers...)

	app.commands = strings.Split(*commands, ",")

	if *genesis != "" {
		h, err := chainhash.NewHashFromStr(*genesis)
		if err == nil {
			app.genesisHash = *h
		}
	}
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
		app)
	if err != nil {
		panic(err)
	}
	app.hostNode = hostNode

	go func() {
		err := app.hostNode.StartDiscovery()
		if err != nil {
			logger.Errorf("error discovering peers: %s", err)
		}
	}()
}

func (app *goTestApp) executeCommands() {
	for _, command := range app.commands {
		app.executeCommand(command)
	}
}

func (app *goTestApp) executeCommand(command string) {
	switch command {
	case "invalidVersionMessage":
		app.commandInvalidVersionMessage()
		break

	case "invalidRejectMessage":
		app.commandInvalidRejectMessage()
		break

	case "invalidRawData":
		app.commandInvalidRawData()
		break

	case "hugeMessageSize":
		app.commandHugeMessageSize()
		break

	default:
		logger.Errorf("Unknown command: %s", command)
		break
	}
}

func (app *goTestApp) commandInvalidVersionMessage() {
	message := &pb.VersionMessage{}
	message.Version = 10000
	message.PeerID = makeRandomBytes(150)
	message.PeerInfo = makeRandomBytes(1)
	message.GenesisHash = makeRandomBytes(200)
	message.Height = 2000000

	app.getPeerNode(0).SendMessage(message)

	logger.Info("commandInvalidVersionMessage finished.")
	logger.Info("Expect 'error handling message: multihash length inconsistent' in the Beacon stdout.")
}

func (app *goTestApp) commandInvalidRejectMessage() {
	message := &pb.RejectMessage{}
	message.Message = "a"

	app.getPeerNode(0).SendMessage(message)

	logger.Info("commandInvalidRejectMessage finished.")
	logger.Info("Expect 'error processing message from peer PEERID: stream reset' in the Beacon stdout.")
}

func (app *goTestApp) commandInvalidRawData() {
	connection := app.getPeerNode(0).GetConnection()
	writer := bufio.NewWriter(connection)
	data := makeRandomBytes(10000)
	writer.Write(data)
	writer.Flush()

	logger.Info("commandInvalidRawData finished.")
	logger.Info("Expect 'processMessages: error when reading message' in the Beacon stdout.")
}

func (app *goTestApp) commandHugeMessageSize() {
	connection := app.getPeerNode(0).GetConnection()
	writer := bufio.NewWriter(connection)

	buf := make([]byte, 4)

	binary.LittleEndian.PutUint32(buf, uint32(0xffffffff))
	writer.Write(buf)
	writer.Write(buf)
	writer.Flush()

	logger.Info("commandHugeMessageSize finished.")
	logger.Info("Expect 'processMessages: message size 4294967295 is too large' in the Beacon stdout.")
}

func (app *goTestApp) getPeerNode(index int) *p2p.Peer {
	peerList := app.hostNode.GetPeerList()
	if index >= len(peerList) {
		logger.Error("Peer index is out of range")
		return nil
	}
	return peerList[index]
}

func (app *goTestApp) disconnectAllPeers() {
	//for len(app.hostNode.GetPeerList()) > 0 {
	//	app.hostNode.DisconnectPeer(app.hostNode.GetPeerList()[0])
	//}
}

func main() {
	logger.SetLevel(logger.TraceLevel)

	logger.Info("Test")

	app := goTestApp{}
	app.run()
}
