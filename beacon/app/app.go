package app

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/beacon/rpc"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"google.golang.org/grpc"
)

// Config is the config of an App
type Config struct {
	P2PAddress           string
	RPCAddress           string
	GenesisTime          uint64
	InitialValidatorList []beacon.InitialValidatorEntry
	NetworkConfig        *config.Config
}

// NewConfig creates a default Config
func NewConfig() Config {
	return Config{
		P2PAddress:           "127.0.0.1:190001",
		RPCAddress:           "127.0.0.1:20002",
		GenesisTime:          uint64(time.Now().Unix()),
		InitialValidatorList: []beacon.InitialValidatorEntry{},
		NetworkConfig:        &config.MainNetConfig,
	}
}

// App contains all the high level states and workflow for P2P module
type App struct {
	config       Config
	privateKey   crypto.PrivKey
	publicKey    crypto.PubKey
	grpcServer   *grpc.Server
	database     *db.InMemoryDB
	blockchain   *beacon.Blockchain
	p2pRPCClient pb.P2PRPCClient
	p2pListener  pb.P2PRPC_ListenForMessagesClient
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
	app.doStateInitialize()
}

// Setup necessary variable
func (app *App) doStateInitialize() {
	app.doStateLoadConfig()
}

// Load user config from configure file
func (app *App) doStateLoadConfig() {
	app.doStateLoadDatabase()
}

func (app *App) doStateLoadDatabase() {
	app.database = db.NewInMemoryDB()

	app.doStateLoadBlockchain()
}

func (app *App) doStateLoadBlockchain() {
	blockchain, err := beacon.NewBlockchainWithInitialValidators(app.database, app.config.NetworkConfig, app.config.InitialValidatorList, true, app.config.GenesisTime)
	if err != nil {
		panic(err)
	}

	app.blockchain = blockchain

	app.doStateConnectP2PRPC()
}

func (app *App) doStateConnectP2PRPC() {
	conn, err := grpc.Dial(app.config.P2PAddress, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	app.p2pRPCClient = pb.NewP2PRPCClient(conn)

	sub, err := app.p2pRPCClient.Subscribe(context.Background(), &pb.SubscriptionRequest{Topic: "block"})
	if err != nil {
		panic(err)
	}

	listener, err := app.p2pRPCClient.ListenForMessages(context.Background(), sub)
	if err != nil {
		panic(err)
	}

	app.p2pListener = listener

	app.doStateCreateRPCServer()
}

func (app *App) doStateCreateRPCServer() {
	err := rpc.Serve(app.config.RPCAddress, app.blockchain, app.p2pRPCClient)
	if err != nil {
		panic(err)
	}

	go app.doMainLoop()
}

func (app *App) doMainLoop() {
	newBlocks := make(chan primitives.Block)
	go func() {
		err := app.blockchain.HandleNewBlocks(newBlocks)
		if err != nil {
			panic(err)
		}
	}()

	for {
		msg, err := app.p2pListener.Recv()
		if err != nil {
			panic(err)
		}

		blockProto := new(pb.Block)

		err = proto.Unmarshal(msg.Data, blockProto)
		if err != nil {
			continue
		}

		block, err := primitives.BlockFromProto(blockProto)
		if err != nil {
			continue
		}

		newBlocks <- *block
	}
}
