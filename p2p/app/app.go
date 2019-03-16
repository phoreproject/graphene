package app

import (
	"crypto/rand"
	"net"
	"time"

	"github.com/phoreproject/synapse/pb"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/p2p/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Config is the config of an P2PApp
type Config struct {
	ListeningAddress   string
	RPCAddress         string
	MinPeerCountToWait int
	HeartBeatInterval  int
	TimeOutInterval    int
	DiscoveryOptions   p2p.DiscoveryOptions
	AddedPeers         []*peerstore.PeerInfo
}

// NewConfig creates a default Config
func NewConfig() Config {
	return Config{
		ListeningAddress:   "/ip4/127.0.0.1/tcp/20000",
		RPCAddress:         "127.0.0.1:20001",
		MinPeerCountToWait: 5,
		HeartBeatInterval:  2 * 60 * 1000,
		TimeOutInterval:    20 * 60 * 1000,
		DiscoveryOptions:   p2p.NewDiscoveryOptions(),
	}
}

// P2PApp contains all the high level states and workflow for P2P module
type P2PApp struct {
	config     Config
	privateKey crypto.PrivKey
	publicKey  crypto.PubKey
	hostNode   *p2p.HostNode
	grpcServer *grpc.Server
}

// NewP2PApp creates a new instance of P2PApp
func NewP2PApp(config Config) *P2PApp {
	app := &P2PApp{
		config: config,
	}
	return app
}

// Initialize initializes the P2PApp by loading the config and setting up networking.
func (app *P2PApp) Initialize() error {
	app.loadConfig()
	app.createHost()
	app.createRPCServer()

	return nil
}

// Run runs the main loop of P2PApp
func (app *P2PApp) Run() error {
	err := app.connectAddedPeers()
	if err != nil {
		return err
	}
	err = app.discoverPeers()
	if err != nil {
		return err
	}
	app.waitPeersReady()
	return nil
}

// GetHostNode gets the host node
func (app *P2PApp) GetHostNode() *p2p.HostNode {
	return app.hostNode
}

// Load user config from configure file
func (app *P2PApp) loadConfig() {
	// TODO: need to load the variables from config file
	privateKey, publicKey, _ := crypto.GenerateSecp256k1Key(rand.Reader)
	app.privateKey = privateKey
	app.publicKey = publicKey
}

func (app *P2PApp) createHost() {
	addr, err := ma.NewMultiaddr(app.config.ListeningAddress)
	if err != nil {
		panic(err)
	}

	hostNode, err := p2p.NewHostNode(addr, app.publicKey, app.privateKey, app.config.DiscoveryOptions)
	if err != nil {
		panic(err)
	}
	app.hostNode = hostNode
}

func (app *P2PApp) createRPCServer() {
	app.grpcServer = grpc.NewServer()
	reflection.Register(app.grpcServer)

	pb.RegisterP2PRPCServer(app.grpcServer, rpc.NewRPCServer(app.hostNode))

	lis, err := net.Listen("tcp", app.config.RPCAddress)
	if err != nil {
		panic(err)
	}
	go func() {
		err = app.grpcServer.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()
}

func (app *P2PApp) connectAddedPeers() error {
	for _, peerInfo := range app.config.AddedPeers {
		_, err := app.hostNode.Connect(*peerInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *P2PApp) discoverPeers() error {
	return app.hostNode.StartDiscovery()
}

func (app *P2PApp) waitPeersReady() {
	for {
		// TODO: the count 5 should be loaded from config file
		if app.hostNode.PeersConnected() >= app.config.MinPeerCountToWait {
			//app.doStateSyncBeaconBlocks()
			break
		}

		time.Sleep(1 * time.Second)
	}
}

func (app *P2PApp) isHostReady() bool {
	return app.hostNode != nil
}
