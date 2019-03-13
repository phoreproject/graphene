package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/beacon/rpc"
	"github.com/phoreproject/synapse/p2p"
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
	needInitSync bool
}

// NewBeaconApp creates a new instance of BeaconApp
func NewBeaconApp(config Config) *BeaconApp {
	app := &BeaconApp{
		config:       config,
		needInitSync: true,
	}
	return app
}

// Run runs the main loop of BeaconApp
func (app *BeaconApp) Run() error {
	err := app.initialize()
	if err != nil {
		return err
	}
	err = app.loadConfig()
	if err != nil {
		return err
	}
	err = app.loadDatabase()
	if err != nil {
		return err
	}
	err = app.loadBlockchain()
	if err != nil {
		return err
	}
	err = app.connectP2PRPC()
	if err != nil {
		return err
	}
	err = app.createRPCServer()
	if err != nil {
		return err
	}

	err = app.registerMessageHandlers()
	if err != nil {
		return err
	}

	err = app.startInitSync()
	if err != nil {
		return err
	}

	return app.runMainLoop()
}

// Setup necessary variable
func (app *BeaconApp) initialize() error {
	return nil
}

// Load user config from configure file
func (app *BeaconApp) loadConfig() error {
	return nil
}

func (app *BeaconApp) loadDatabase() error {
	dir, err := config.GetBaseDirectory(true)
	if err != nil {
		panic(err)
	}

	dbDir := filepath.Join(dir, "db")
	dbValuesDir := filepath.Join(dir, "dbv")

	err = os.MkdirAll(dbDir, 0777)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(dbValuesDir, 0777)
	if err != nil {
		panic(err)
	}

	logger.Info("initializing client")

	logger.Info("initializing database")
	database := db.NewBadgerDB(dbDir, dbValuesDir)

	if app.config.Resync {
		logger.Info("dropping all keys in database to resync")
		err := database.Flush()
		if err != nil {
			return err
		}
	}

	if t, err := database.GetGenesisTime(); err == nil {
		fmt.Println(t, err == nil)
		logger.Debug("using time from database")
		app.genesisTime = t
	} else {
		logger.WithField("genesisTime", app.config.GenesisTime).Debug("using time from config")
		err := database.SetGenesisTime(app.config.GenesisTime)
		if err != nil {
			return err
		}
		app.genesisTime = app.config.GenesisTime
	}

	app.database = database

	return nil
}

func (app *BeaconApp) loadBlockchain() error {
	blockchain, err := beacon.NewBlockchainWithInitialValidators(app.database, app.config.NetworkConfig, app.config.InitialValidatorList, true, app.genesisTime)
	if err != nil {
		panic(err)
	}

	app.blockchain = blockchain

	return nil
}

func (app *BeaconApp) connectP2PRPC() error {
	conn, err := grpc.Dial(app.config.P2PAddress, grpc.WithInsecure())
	if err != nil {
		return err
	}

	app.p2pRPCClient = pb.NewP2PRPCClient(conn)

	sub, err := app.p2pRPCClient.Subscribe(context.Background(), &pb.SubscriptionRequest{Topic: "block"})
	if err != nil {
		return err
	}

	listener, err := app.p2pRPCClient.ListenForMessages(context.Background(), sub)
	if err != nil {
		return err
	}

	app.p2pListener = listener

	return nil
}

func (app *BeaconApp) createRPCServer() error {
	go func() {
		err := rpc.Serve(app.config.RPCAddress, app.blockchain, app.p2pRPCClient)
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

func (app *BeaconApp) startInitSync() error {
	go func() {
		for {
			// nest for..loop to allow sleep at the end of the outter loop
			for {
				peerListResponse, err := app.p2pRPCClient.GetPeers(context.Background(), &empty.Empty{})
				if err != nil {
					break
				}

				peerList := peerListResponse.Peers
				if len(peerList) == 0 {
					break
				}

				peerID := peerList[0].PeerID

				headHash, err := app.database.GetHeadBlock()
				if err != nil {
					break
				}

				message := &pb.GetBlockMessage{}
				message.LocatorHahes = make([][]byte, 1)
				message.LocatorHahes[0] = headHash.CloneBytes()

				data, err := p2p.MessageToBytes(message)
				if err != nil {
					break
				}
				app.p2pRPCClient.SendDirectMessage(context.Background(), &pb.SendDirectMessageRequest{
					PeerID:  peerID,
					Message: data,
				})

				logger.Debugf("Start init sync with %s", peerID)

				return
			}

			time.Sleep(10 * time.Millisecond)
		}
	}()

	return nil
}

func (app *BeaconApp) runMainLoop() error {
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
	return app.waitForExit()
}

func (app BeaconApp) waitForExit() error {
	defer app.exit()

	signalHandler := make(chan os.Signal, 1)
	signal.Notify(signalHandler, os.Interrupt)
	<-signalHandler

	logger.Info("exiting")
	return nil
}

func (app BeaconApp) registerMessageHandlers() error {
	err := app.registerDirectMessage("pb.GetBlockMessage", app.onMessageGetBlock)
	if err != nil {
		return err
	}

	err = app.registerDirectMessage("pb.BlockMessage", app.onMessageBlock)
	if err != nil {
		return err
	}

	return nil
}

func (app BeaconApp) registerDirectMessage(messageName string, handler func(string, proto.Message)) error {
	ctx := context.Background()
	{
		sub, err := app.p2pRPCClient.SubscribeDirectMessage(ctx, &pb.SubscribeDirectMessageRequest{
			MessageName: messageName,
		})
		if err != nil {
			return err
		}
		listener, err := app.p2pRPCClient.ListenForDirectMessages(ctx, sub)
		if err != nil {
			return err
		}
		go func() {
			for {
				msg, err := listener.Recv()
				if err != nil {
					panic(err)
				}
				m, err := p2p.BytesToMessage(msg.Data)
				if err != nil {
					panic(err)
				}
				handler(msg.PeerID, m)
			}
		}()
	}

	return nil
}

func (app BeaconApp) onMessageGetBlock(peerID string, message proto.Message) {
	blockMessage := &pb.BlockMessage{}
	data, err := p2p.MessageToBytes(blockMessage)
	if err != nil {
		return
	}
	app.p2pRPCClient.SendDirectMessage(context.Background(), &pb.SendDirectMessageRequest{
		PeerID:  peerID,
		Message: data,
	})
}

func (app BeaconApp) onMessageBlock(peerID string, message proto.Message) {
	logger.Debug("Received block")
}

func (app BeaconApp) exit() {
	app.database.Close()
}
