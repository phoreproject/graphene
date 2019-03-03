package app

import (
	"crypto/rand"
	"net"
	"time"

	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/utils"

	"github.com/golang/protobuf/proto"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/p2p"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Config is the config of an App
type Config struct {
	ListeningAddress   string
	RPCAddress         string
	MinPeerCountToWait int
	HeartBeatInterval  int
	TimeOutInterval    int
	DiscoveryOptions   *p2p.DiscoveryOptions
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

// App contains all the high level states and workflow for P2P module
type App struct {
	config     Config
	privateKey crypto.PrivKey
	publicKey  crypto.PubKey
	hostNode   *p2p.HostNode
	grpcServer *grpc.Server
}

// NewApp creates a new instance of App
func NewApp(config Config) *App {
	app := &App{
		config: config,
	}
	return app
}

// Run runs the main loop of App
func (app *App) Run() {
	go app.doMainLoop()
	app.doStateInitialize()
}

// GetHostNode gets the host node
func (app *App) GetHostNode() *p2p.HostNode {
	return app.hostNode
}

// Setup necessary variable
func (app *App) doStateInitialize() {
	app.doStateLoadConfig()
}

// Load user config from configure file
func (app *App) doStateLoadConfig() {
	// TODO: need to load the variables from config file
	privateKey, publicKey, _ := crypto.GenerateSecp256k1Key(rand.Reader)
	app.privateKey = privateKey
	app.publicKey = publicKey

	app.doStateCreateHost()
}

func (app *App) doStateCreateHost() {
	addr, err := ma.NewMultiaddr(app.config.ListeningAddress)
	if err != nil {
		panic(err)
	}

	hostNode, err := p2p.NewHostNode(addr, app.publicKey, app.privateKey)
	if err != nil {
		panic(err)
	}
	hostNode.SetOnPeerConnectedHandler(app.onPeerConnected)
	app.hostNode = hostNode

	app.registerMessageHandlers()

	app.doStateCreateRPCServer()
}

func (app *App) doStateCreateRPCServer() {
	app.grpcServer = grpc.NewServer()
	reflection.Register(app.grpcServer)

	pb.RegisterP2PRPCServer(app.grpcServer, p2p.NewRPCServer(app.hostNode))

	lis, err := net.Listen("tcp", app.config.RPCAddress)
	if err != nil {
		panic(err)
	}
	err = app.grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}

	app.doStateConnectAddedPeers()
}

func (app *App) doStateConnectAddedPeers() {
	for _, peerInfo := range app.config.AddedPeers {
		app.hostNode.Connect(peerInfo)
	}

	app.doStateDiscoverPeers()
}

func (app *App) doStateDiscoverPeers() {
	p2p.StartDiscovery(app.hostNode, app.config.DiscoveryOptions)

	app.doStateWaitPeersReady()
}

func (app *App) doStateWaitPeersReady() {
	for {
		// TODO: the count 5 should be loaded from config file
		if len(app.hostNode.GetLivePeerList()) >= app.config.MinPeerCountToWait {
			//app.doStateSyncBeaconBlocks()
			break
		}
	}
}

func (app *App) onPeerConnected(peer *p2p.PeerNode) {
	peer.SendMessage(&pb.VersionMessage{
		Version: 0,
	})
}

func (app *App) registerMessageHandlers() {
	app.hostNode.SetAnyMessageHandler(app.onAnyMessage)

	app.hostNode.RegisterMessageHandler("pb.VersionMessage", app.onMessageVersion)
	app.hostNode.RegisterMessageHandler("pb.VerackMessage", app.onMessageVerack)
	app.hostNode.RegisterMessageHandler("pb.GetBlockMessage", app.onMessageGetBlock)
	app.hostNode.RegisterMessageHandler("pb.BlockMessage", app.onMessageBlock)
	app.hostNode.RegisterMessageHandler("pb.PingMessage", app.onMessagePing)
	app.hostNode.RegisterMessageHandler("pb.PongMessage", app.onMessagePong)
}

func (app *App) onMessageVersion(peer *p2p.PeerNode, message proto.Message) {
	logger.Debug("Received version")

	peer.SendMessage(&pb.VerackMessage{})
}

func (app *App) onMessageVerack(peer *p2p.PeerNode, message proto.Message) {
	logger.Debug("Received verack")

	app.hostNode.PeerDoneHandShake(peer)
}

func (app *App) onMessageGetBlock(peer *p2p.PeerNode, message proto.Message) {
	blockMessage := &pb.BlockMessage{}
	peer.SendMessage(blockMessage)
}

func (app *App) onMessageBlock(peer *p2p.PeerNode, message proto.Message) {
	logger.Debug("Received block")
}

func (app *App) onMessagePing(peer *p2p.PeerNode, message proto.Message) {
	peer.SendMessage(&pb.PongMessage{
		Nonce: message.(*pb.PingMessage).Nonce,
	})
}

func (app *App) onMessagePong(peer *p2p.PeerNode, message proto.Message) {
	if peer.LastPingNonce == message.(*pb.PongMessage).Nonce {
	}
}

func (app *App) onAnyMessage(peer *p2p.PeerNode, message proto.Message) bool {
	peer.LastMessageTime = utils.GetCurrentMilliseconds()

	return true
}

func (app *App) doMainLoop() {
	for {
		app.doHeartBeat()

		time.Sleep(100 * time.Millisecond)
	}
}

func (app *App) doHeartBeat() {
	if !app.isHostReady() {
		return
	}

	heartBeatInterval := uint64(app.config.HeartBeatInterval)
	timeOutInterval := uint64(app.config.TimeOutInterval)
	currentTime := utils.GetCurrentMilliseconds()

	cotinueChecking := true

	for cotinueChecking {
		cotinueChecking = false

		for _, peer := range app.hostNode.GetLivePeerList() {
			if peer.LastPingTime > 0 && currentTime > peer.LastPingTime+timeOutInterval {
				// time out, drop the peer
				app.hostNode.DisconnectPeer(peer)
				// DisconnectPeer will pollute live peer list and we can't continue the loop
				// let's restart over
				cotinueChecking = true
				break
			} else if currentTime > peer.LastMessageTime+heartBeatInterval || currentTime > peer.LastPingTime+heartBeatInterval {
				peer.LastPingTime = currentTime
				peer.LastMessageTime = currentTime
				peer.LastPingNonce = 1
				peer.SendMessage(&pb.PingMessage{
					Nonce: peer.LastPingNonce,
				})
			}
		}
	}
}

func (app *App) isHostReady() bool {
	return app.hostNode != nil
}
