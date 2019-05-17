package app

import (
	"crypto/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	homedir "github.com/mitchellh/go-homedir"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/beacon/rpc"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/primitives"
	logger "github.com/sirupsen/logrus"
)

// Config is the config of an BeaconApp
type Config struct {
	RPCAddress             string
	GenesisTime            uint64
	DataDirectory          string
	InitialValidatorList   []beacon.InitialValidatorEntry
	NetworkConfig          *config.Config
	Resync                 bool
	IsIntegrationTest      bool
	InitialSyncConnections int
	ListeningAddress       string
	MinPeerCountToWait     int
	HeartBeatInterval      time.Duration
	TimeOutInterval        time.Duration
	DiscoveryOptions       p2p.DiscoveryOptions
	MaxPeers               int
}

// NewConfig creates a default Config
func NewConfig() Config {
	return Config{
		ListeningAddress:       "/ip4/127.0.0.1/tcp/20000",
		RPCAddress:             "127.0.0.1:20002",
		GenesisTime:            uint64(time.Now().Unix()),
		InitialValidatorList:   []beacon.InitialValidatorEntry{},
		NetworkConfig:          &config.MainNetConfig,
		IsIntegrationTest:      false,
		DataDirectory:          "",
		InitialSyncConnections: 1,
		MinPeerCountToWait:     1,
		HeartBeatInterval:      8 * time.Second,
		TimeOutInterval:        16 * time.Second,
		DiscoveryOptions:       p2p.NewDiscoveryOptions(),
		MaxPeers:               16,
	}
}

// BeaconApp contains all the high level states and workflow for P2P module
type BeaconApp struct {
	// config is the config passed to the app.
	config Config

	// exitChan receives a struct when an exit is requested.
	exitChan chan struct{}

	database   db.Database
	blockchain *beacon.Blockchain
	mempool    beacon.Mempool

	// P2P
	hostNode    *p2p.HostNode
	syncManager beacon.SyncManager
}

// NewBeaconApp creates a new instance of BeaconApp
func NewBeaconApp(config Config) *BeaconApp {
	app := &BeaconApp{
		config:   config,
		exitChan: make(chan struct{}),
		mempool:  beacon.NewMempool(),
	}
	return app
}

// Run runs the main loop of BeaconApp
func (app *BeaconApp) Run() error {
	err := app.loadConfig()
	if err != nil {
		return err
	}
	err = app.loadDatabase()
	if err != nil {
		return err
	}
	err = app.loadP2P()
	if err != nil {
		return err
	}

	signalHandler := make(chan os.Signal, 1)
	signal.Notify(signalHandler, os.Interrupt, syscall.SIGTERM)

	if !app.config.IsIntegrationTest {
		go app.listenForInterrupt(signalHandler)
	}

	err = app.loadBlockchain()
	if err != nil {
		return err
	}
	err = app.createRPCServer()
	if err != nil {
		return err
	}

	app.syncManager = beacon.NewSyncManager(app.hostNode, app.blockchain)

	app.syncManager.Start()

	return app.runMainLoop()
}

func (app *BeaconApp) getHostKey() (crypto.PrivKey, crypto.PubKey, error) {
	var pub crypto.PubKey
	k, err := app.database.GetHostKey()

	if err != nil {
		logger.Debug("private key not found, generating...")
		k, pub, err = crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return nil, nil, err
		}

		err := app.database.SetHostKey(k)
		if err != nil {
			return nil, nil, err
		}
	}

	if pub == nil {
		pub = k.GetPublic()
	}

	return k, pub, nil
}

func (app *BeaconApp) loadP2P() error {
	logger.Info("loading P2P")
	addr, err := ma.NewMultiaddr(app.config.ListeningAddress)
	if err != nil {
		panic(err)
	}

	priv, pub, err := app.getHostKey()
	if err != nil {
		panic(err)
	}

	hostNode, err := p2p.NewHostNode(addr, pub, priv, app.config.DiscoveryOptions, app.config.TimeOutInterval, app.config.MaxPeers, app.config.HeartBeatInterval)
	if err != nil {
		panic(err)
	}
	app.hostNode = hostNode

	logger.Debug("starting peer discovery")
	go app.hostNode.StartDiscovery()

	return nil
}

// GetHostNode gets the host node
func (app *BeaconApp) GetHostNode() *p2p.HostNode {
	return app.hostNode
}

// Load user config from configure file
func (app *BeaconApp) loadConfig() error {
	return nil
}

func (app *BeaconApp) loadDatabase() error {
	var dir string
	if app.config.DataDirectory == "" {
		dataDir, err := config.GetBaseDirectory(true)
		if err != nil {
			panic(err)
		}
		dir = dataDir
	} else {
		d, err := homedir.Expand(app.config.DataDirectory)
		if err != nil {
			panic(err)
		}
		dir = d
	}

	err := os.MkdirAll(dir, 0777)
	if err != nil {
		panic(err)
	}

	logger.Info("initializing client")

	logger.Info("initializing database")
	database := db.NewBadgerDB(dir)

	if app.config.Resync {
		logger.Info("dropping all keys in database to resync")

		key, err := database.GetHostKey()
		if err != nil {
			key = nil
		}

		err = database.Flush()
		if err != nil {
			return err
		}

		if key != nil {
			err := database.SetHostKey(key)
			if err != nil {
				panic(err)
			}
		}
	}

	app.database = database

	return nil
}

func (app *BeaconApp) loadBlockchain() error {
	var genesisTime uint64
	if t, err := app.database.GetGenesisTime(); err == nil {
		logger.WithField("genesisTime", t).Info("using time from database")
		genesisTime = t
	} else {
		logger.WithField("genesisTime", app.config.GenesisTime).Info("using time from config")
		err := app.database.SetGenesisTime(app.config.GenesisTime)
		if err != nil {
			return err
		}
		genesisTime = app.config.GenesisTime
	}

	blockchain, err := beacon.NewBlockchainWithInitialValidators(app.database, app.config.NetworkConfig, app.config.InitialValidatorList, true, genesisTime)
	if err != nil {
		panic(err)
	}

	app.blockchain = blockchain

	go app.watchBlocksForMempool()

	return nil
}

func (app *BeaconApp) createRPCServer() error {
	go func() {
		err := rpc.Serve(app.config.RPCAddress, app.blockchain, app.hostNode, &app.mempool)
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

// WaitForConnections waits until beacon app is connected
func (app *BeaconApp) WaitForConnections(numConnections int) {
	for {
		if app.hostNode.PeersConnected() >= numConnections {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (app *BeaconApp) runMainLoop() error {
	go func() {
		app.WaitForConnections(app.config.MinPeerCountToWait)

		go app.syncManager.TryInitialSync()

		go app.syncManager.ListenForBlocks()
	}()

	// the main loop for this thread is waiting for the exit and cleaning up
	app.waitForExit()

	return nil
}

func (app BeaconApp) listenForInterrupt(signalHandler chan os.Signal) {
	<-signalHandler

	app.exitChan <- struct{}{}
}

func (app BeaconApp) waitForExit() {
	<-app.exitChan

	app.exit()

	logger.Info("exiting")
}

func (app BeaconApp) exit() {
	err := app.database.Close()
	if err != nil {
		panic(err)
	}

	for _, p := range app.hostNode.GetPeerList() {
		p.Disconnect()
	}

	os.Exit(0)
}

// Exit sends a request to exit the application.
func (app BeaconApp) Exit() {
	app.exitChan <- struct{}{}
}

// watchBlocksForMempool watches for blocks and removes the corresponding attestations from the mempool.
func (app BeaconApp) watchBlocksForMempool() {
	for {
		bl := <-app.blockchain.ConnectBlockNotifier.Watch()

		block := bl.(primitives.Block)

		for _, a := range block.BlockBody.Attestations {
			app.mempool.RemoveAttestationsFromBitfield(a.Data.Slot, a.Data.Shard, a.ParticipationBitfield)
		}
	}
}
