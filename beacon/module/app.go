package module

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/phoreproject/synapse/primitives"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/mitchellh/go-homedir"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/beacon/rpc"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/utils"
	logger "github.com/sirupsen/logrus"
)

// Config is the config of an BeaconApp
type Config struct {
	RPCAddress             string
	DataDirectory          string
	NetworkConfig          *config.Config
	Resync                 bool
	IsIntegrationTest      bool
	InitialSyncConnections int
	ListeningAddress       string
	MinPeerCountToWait     int
	HeartBeatInterval      time.Duration
	TimeOutInterval        time.Duration
	MaxPeers               int

	// These options are filled in through the chain file.
	GenesisTime          uint64
	InitialValidatorList []primitives.InitialValidatorEntry
	DiscoveryOptions     p2p.DiscoveryOptions
	LogLevel             logger.Level
}

// NewConfig creates a default Config
func NewConfig() Config {
	return Config{
		ListeningAddress:       "/ip4/127.0.0.1/tcp/20000",
		RPCAddress:             "/ip4/127.0.0.1/tcp/20002",
		GenesisTime:            uint64(utils.Now().Unix()),
		InitialValidatorList:   []primitives.InitialValidatorEntry{},
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
	exited   *sync.Mutex

	database   db.Database
	blockchain *beacon.Blockchain
	mempool    *beacon.Mempool

	// P2P
	hostNode    *p2p.HostNode
	syncManager beacon.SyncManager
}

// NewBeaconApp creates a new instance of BeaconApp
func NewBeaconApp(options config.Options) (*BeaconApp, error) {
	initialPeers, err := p2p.ParseInitialConnections(options.InitialConnections)
	if err != nil {
		logger.Fatal(err)
	}

	f, err := os.Open(options.ChainCFG)
	if err != nil {
		logger.Fatal(err)
	}

	beaconConfig, err := ReadChainFileToConfig(f)
	if err != nil {
		logger.Fatal(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	beaconConfig.ListeningAddress = options.P2PListen
	beaconConfig.RPCAddress = options.RPCListen
	beaconConfig.DiscoveryOptions.PeerAddresses = append(beaconConfig.DiscoveryOptions.PeerAddresses, initialPeers...)
	beaconConfig.DataDirectory = options.DataDir

	beaconConfig.Resync = options.Resync
	if beaconConfig.GenesisTime == 0 {
		if options.GenesisTime == "" {
			beaconConfig.GenesisTime = uint64(utils.Now().Unix())
		} else {
			genesisTimeString := options.GenesisTime
			if strings.HasPrefix(genesisTimeString, "+") {
				offsetString := genesisTimeString[1:]

				offset, err := strconv.Atoi(offsetString)
				if err != nil {
					panic(fmt.Errorf("invalid offset (should be number): %s", offsetString))
				}

				beaconConfig.GenesisTime = uint64(utils.Now().Add(time.Duration(offset) * time.Second).Unix())
			} else if strings.HasPrefix(genesisTimeString, "-") {
				offsetString := genesisTimeString[1:]

				offset, err := strconv.Atoi(offsetString)
				if err != nil {
					panic(fmt.Errorf("invalid offset (should be number): %s", offsetString))
				}

				beaconConfig.GenesisTime = uint64(utils.Now().Add(time.Duration(-offset) * time.Second).Unix())
			} else {
				offset, err := strconv.Atoi(genesisTimeString)
				if err != nil {
					panic(fmt.Errorf("invalid genesis time (should be number): %s", genesisTimeString))
				}

				beaconConfig.GenesisTime = uint64(offset)
			}
		}
	}

	app := &BeaconApp{
		config:   *beaconConfig,
		exitChan: make(chan struct{}),
		exited:   new(sync.Mutex),
	}

	err = app.loadConfig()
	if err != nil {
		return nil, err
	}
	err = app.loadDatabase()
	if err != nil {
		return nil, err
	}

	signalHandler := make(chan os.Signal, 1)
	signal.Notify(signalHandler, os.Interrupt, syscall.SIGTERM)

	if !app.config.IsIntegrationTest {
		go app.listenForInterrupt(signalHandler)
	}

	err = app.loadBlockchain()
	if err != nil {
		return nil, err
	}

	err = app.loadP2P()
	if err != nil {
		return nil, err
	}
	err = app.createRPCServer()
	if err != nil {
		return nil, err
	}
	app.syncManager = beacon.NewSyncManager(app.hostNode, app.blockchain, app.mempool)

	// locked while running
	app.exited.Lock()
	return app, nil
}

// Run runs the main loop of BeaconApp
func (app *BeaconApp) Run() error {
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

	hostNode, err := p2p.NewHostNode(addr, pub, priv, app.config.DiscoveryOptions, app.config.TimeOutInterval, app.config.MaxPeers, app.config.HeartBeatInterval, app.blockchain)
	if err != nil {
		panic(err)
	}
	app.hostNode = hostNode

	logger.Debug("starting peer discovery")
	go func() {
		err := app.hostNode.StartDiscovery()
		if err != nil {
			logger.Errorf("error discovering peers: %s", err)
		}
	}()

	return nil
}

// GetHostNode gets the host node
func (app *BeaconApp) GetHostNode() *p2p.HostNode {
	return app.hostNode
}

// GetBlockchain gets the block chain
func (app *BeaconApp) GetBlockchain() *beacon.Blockchain {
	return app.blockchain
}

// GetSyncManager gets the syncManager
func (app *BeaconApp) GetSyncManager() *beacon.SyncManager {
	return &app.syncManager
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

	app.mempool = beacon.NewMempool(blockchain)
	return nil
}

func (app *BeaconApp) createRPCServer() error {
	rpcListen, err := ma.NewMultiaddr(app.config.RPCAddress)
	if err != nil {
		return err
	}

	rpcListenAddr, err := manet.ToNetAddr(rpcListen)
	if err != nil {
		return err
	}

	go func() {
		err := rpc.Serve(rpcListenAddr.Network(), rpcListenAddr.String(), app.blockchain, app.hostNode, app.mempool)
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

		go func() {
			err := app.syncManager.ListenForBlocks()
			if err != nil {
				logger.Errorf("error listening for blocks: %s", err)
			}
		}()

		go func() {
			err := app.syncManager.ListenForAttestations()
			if err != nil {
				logger.Errorf("error listening for attestations: %s", err)
			}
		}()
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

	app.exited.Unlock()

	os.Exit(0)
}

// Exit sends a request to exit the application.
func (app BeaconApp) Exit() {
	app.exitChan <- struct{}{}
}

// WaitForExit waits for the beacon chain to exit.
func (app BeaconApp) WaitForExit() {
	app.exited.Lock()
	defer app.exited.Unlock()
}
