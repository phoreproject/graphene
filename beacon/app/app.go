package app

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
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

	logger "github.com/sirupsen/logrus"
)

// Config is the config of an BeaconApp
type Config struct {
	P2PAddress           string
	RPCAddress           string
	GenesisTime          uint64
	InitialValidatorList []beacon.InitialValidatorEntry
	NetworkConfig        *config.Config
	Resync               bool
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

// BeaconApp contains all the high level states and workflow for P2P module
type BeaconApp struct {
	config       Config
	privateKey   crypto.PrivKey
	publicKey    crypto.PubKey
	grpcServer   *grpc.Server
	database     db.Database
	blockchain   *beacon.Blockchain
	p2pRPCClient pb.P2PRPCClient
	p2pListener  pb.P2PRPC_ListenForMessagesClient
	genesisTime  uint64
}

// NewBeaconApp creates a new instance of BeaconApp
func NewBeaconApp(config Config) *BeaconApp {
	app := &BeaconApp{
		config: config,
	}
	return app
}

const (
	stateInitialize = iota
	stateLoadConfig
	stateLoadDatabase
	stateLoadBlockchain
	stateConnectP2PRPC
	stateCreateRPCServer
	stateMainLoop
)

func (app *BeaconApp) transitState(state int) {
	switch state {
	case stateInitialize:
		app.initialize()

	case stateLoadConfig:
		app.loadConfig()

	case stateLoadDatabase:
		app.loadDatabase()

	case stateLoadBlockchain:
		app.loadBlockchain()

	case stateConnectP2PRPC:
		app.connectP2PRPC()

	case stateCreateRPCServer:
		app.createRPCServer()

	case stateMainLoop:
		app.runMainLoop()

	default:
		panic("Unknow state")
	}
}

// Run runs the main loop of BeaconApp
func (app *BeaconApp) Run() error {
	app.transitState(stateInitialize)
	app.initialize()
	return nil
}

// Setup necessary variable
func (app *BeaconApp) initialize() {
	app.transitState(stateLoadConfig)
}

// Load user config from configure file
func (app *BeaconApp) loadConfig() {
	app.transitState(stateLoadDatabase)
}

func (app *BeaconApp) loadDatabase() error {
	dir, err := config.GetBaseDirectory(true)
	if err != nil {
		panic(err)
	}

	dbDir := filepath.Join(dir, "db")
	dbValuesDir := filepath.Join(dir, "dbv")

	os.MkdirAll(dbDir, 0777)
	os.MkdirAll(dbValuesDir, 0777)

	logger.Info("initializing client")

	logger.Info("initializing database")
	database := db.NewBadgerDB(dbDir, dbValuesDir)

	if time, err := database.GetGenesisTime(); err != nil {
		logger.Debug("using time from database")
		app.genesisTime = time
	} else {
		logger.Debug("using time from config")
		err := database.SetGenesisTime(app.config.GenesisTime)
		if err != nil {
			return err
		}
	}

	if app.config.Resync {
		logger.Info("dropping all keys in database to resync")
		err := database.Flush()
		if err != nil {
			return err
		}
	}

	defer database.Close()

	app.database = database

	app.transitState(stateLoadBlockchain)

	return nil
}

func (app *BeaconApp) loadBlockchain() {
	blockchain, err := beacon.NewBlockchainWithInitialValidators(app.database, app.config.NetworkConfig, app.config.InitialValidatorList, true, app.genesisTime)
	if err != nil {
		panic(err)
	}

	app.blockchain = blockchain

	app.transitState(stateConnectP2PRPC)
}

func (app *BeaconApp) connectP2PRPC() {
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

	app.transitState(stateCreateRPCServer)
}

func (app *BeaconApp) createRPCServer() {
	go func() {
		err := rpc.Serve(app.config.RPCAddress, app.blockchain, app.p2pRPCClient)
		if err != nil {
			panic(err)
		}
	}()

	app.transitState(stateMainLoop)
}

func (app *BeaconApp) runMainLoop() {
	go func() {
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
	}()

	// the main loop for this thread is waiting for the exit and cleaning up
	app.waitForExit()
}

func (app BeaconApp) waitForExit() {
	defer app.exit()

	signalHandler := make(chan os.Signal, 1)
	signal.Notify(signalHandler, os.Interrupt)
	<-signalHandler

	logger.Info("exiting")
}

func (app BeaconApp) exit() {
	app.database.Close()
}
