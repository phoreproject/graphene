package app

import (
	"crypto/rand"
	"fmt"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"

	"github.com/golang/protobuf/proto"
	crypto "github.com/libp2p/go-libp2p-crypto"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/p2p"
	logger "github.com/sirupsen/logrus"
)

// App contains all the high level states and workflow for Beacon module
type App struct {
	p2pListenPort    int
	privateKey       crypto.PrivKey
	publicKey        crypto.PubKey
	database         *db.InMemoryDB
	discoveryOptions *p2p.DiscoveryOptions
	blockchain       *beacon.Blockchain
	hostNode         *p2p.HostNode
}

var app *App

// NewApp creates a new instance of App
func NewApp() *App {
	if app != nil {
		panic("Beacon App can have only one instance")
	}
	app = &App{}
	return app
}

// GetApp returns the global instance of App
// TODO: not sure if the global instance is needed yet.
func GetApp() *App {
	if app == nil {
		return NewApp()
	}
	return app
}

// Run runs the main loop of App
func (app *App) Run() {
	app.doStateInitialize()
}

func (app *App) gotoStep(f func()) {
	f()
}

// Setup necessary variable
func (app *App) doStateInitialize() {
	app.gotoStep(app.doStateLoadConfig)
}

// Load user config from configure file
func (app *App) doStateLoadConfig() {
	// TODO: need to load the variables from config file
	app.p2pListenPort = 20000
	privateKey, publicKey, _ := crypto.GenerateSecp256k1Key(rand.Reader)
	app.privateKey = privateKey
	app.publicKey = publicKey
	app.discoveryOptions = p2p.NewDiscoveryOptions()

	app.gotoStep(app.doStateLoadDatabase)
}

// Open and load local database
func (app *App) doStateLoadDatabase() {
	app.database = db.NewInMemoryDB()

	app.gotoStep(app.doStateLoadBlockchain)
}

// Open and load the Beacon blockchain
func (app *App) doStateLoadBlockchain() {
	c := config.MainNetConfig
	validators := []beacon.InitialValidatorEntry{}
	/*
		for i := 0; i <= int(c.EpochLength)*(c.TargetCommitteeSize*2); i++ {
			validators = append(validators, beacon.InitialValidatorEntry{
				PubKey:                [96]byte{},
				ProofOfPossession:     [48]byte{},
				WithdrawalShard:       1,
				WithdrawalCredentials: chainhash.Hash{},
			})
		}
	*/
	blockchain, err := beacon.NewBlockchainWithInitialValidators(app.database, &c, validators, true)
	if err != nil {
		panic(err)
	}
	app.blockchain = blockchain

	app.gotoStep(app.doStateCreateHost)
}

// Create P2P host node
func (app *App) doStateCreateHost() {
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", app.p2pListenPort))
	if err != nil {
		panic(err)
	}

	hostNode, err := p2p.NewHostNode(addr, app.publicKey, app.privateKey, app.blockchain)
	if err != nil {
		panic(err)
	}
	hostNode.SetOnPeerConnectedHandler(app.onPeerConnected)
	app.hostNode = hostNode

	app.registerMessageHandlers()

	app.gotoStep(app.doStateDiscoverPeers)
}

// Discover P2P peer nodes
func (app *App) doStateDiscoverPeers() {
	p2p.StartDiscovery(app.hostNode, app.discoveryOptions)

	for {
		// TODO: the count 5 should be loaded from config file
		if len(app.hostNode.GetPeerList()) >= 5 {
			app.gotoStep(app.doStateSyncBeaconBlocks)
			break
		}
	}
}

// Synchronize the blockchain from peers
func (app *App) doStateSyncBeaconBlocks() {
	peerList := app.hostNode.GetPeerList()

	for _, peer := range peerList {
		locatorHash := chainhash.Hash{}
		hashStop := chainhash.Hash{}
		message := &pb.GetBlockMessage{}
		message.LocatorHahes = make([][]byte, 1)
		message.LocatorHahes[0] = locatorHash.CloneBytes()
		message.HashStop = hashStop.CloneBytes()
		peer.SendMessage(message)
	}
}

func (app *App) onPeerConnected(peer *p2p.PeerNode) {
	// TODO: need to handshake with the peer (exchange keys, etc)
}

func (app *App) registerMessageHandlers() {
	app.hostNode.RegisterMessageHandler("pb.GetBlockMessage", app.onMessageGetBlock)
	app.hostNode.RegisterMessageHandler("pb.BlockMessage", app.onMessageBlock)
}

func (app *App) onMessageGetBlock(peer *p2p.PeerNode, message proto.Message) {
	blockMessage := &pb.BlockMessage{}
	peer.SendMessage(blockMessage)
}

func (app *App) onMessageBlock(peer *p2p.PeerNode, message proto.Message) {
	logger.Debug("Received block")
}
