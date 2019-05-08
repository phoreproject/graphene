package explorer

import (
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p-crypto"
	"io"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/mitchellh/go-homedir"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/p2p"

	logger "github.com/sirupsen/logrus"

	"github.com/labstack/echo"
)

// Config is the explorer app config.
type Config struct {
	GenesisTime          uint64
	DataDirectory        string
	InitialValidatorList []beacon.InitialValidatorEntry
	NetworkConfig        *config.Config
	Resync               bool
	ListeningAddress     string
	DiscoveryOptions     p2p.DiscoveryOptions
}

// Explorer is a blockchain explorer.
// The explorer streams blocks from the beacon chain as they are received
// and then keeps track of its own blockchain so that it can access more
// info like forking.
type Explorer struct {
	blockchain *beacon.Blockchain

	// P2P
	hostNode    *p2p.HostNode
	syncManager beacon.SyncManager

	config Config

	database *Database
	chainDB  db.Database
}

// NewExplorer creates a new block explorer
func NewExplorer(c Config, gormDB *gorm.DB) (*Explorer, error) {
	return &Explorer{
		database: NewDatabase(gormDB),
		config:   c,
	}, nil
}

func (ex *Explorer) loadDatabase() error {
	var dir string
	if ex.config.DataDirectory == "" {
		dataDir, err := config.GetBaseDirectory(true)
		if err != nil {
			panic(err)
		}
		dir = dataDir
	} else {
		d, err := homedir.Expand(ex.config.DataDirectory)
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

	if ex.config.Resync {
		logger.Info("dropping all keys in database to resync")
		err := database.Flush()
		if err != nil {
			return err
		}
	}

	ex.chainDB = database

	return nil
}

func (ex *Explorer) loadP2P() error {
	logger.Info("loading P2P")
	addr, err := ma.NewMultiaddr(ex.config.ListeningAddress)
	if err != nil {
		panic(err)
	}

	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic(err)
	}

	hostNode, err := p2p.NewHostNode(addr, pub, priv, ex.config.DiscoveryOptions, 16*time.Second)
	if err != nil {
		panic(err)
	}
	ex.hostNode = hostNode

	logger.Debug("starting peer discovery")
	err = ex.hostNode.StartDiscovery()
	if err != nil {
		panic(err)
	}

	return nil
}

func (ex *Explorer) loadBlockchain() error {
	var genesisTime uint64
	if t, err := ex.chainDB.GetGenesisTime(); err == nil {
		logger.WithField("genesisTime", t).Info("using time from database")
		genesisTime = t
	} else {
		logger.WithField("genesisTime", ex.config.GenesisTime).Info("using time from config")
		err := ex.chainDB.SetGenesisTime(ex.config.GenesisTime)
		if err != nil {
			return err
		}
		genesisTime = ex.config.GenesisTime
	}

	blockchain, err := beacon.NewBlockchainWithInitialValidators(ex.chainDB, ex.config.NetworkConfig, ex.config.InitialValidatorList, true, genesisTime)
	if err != nil {
		panic(err)
	}

	ex.blockchain = blockchain

	return nil
}

// Template is the template engine used by the explorer.
type Template struct {
	templates *template.Template
}

// Render renders the template.
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// IndexSlotData is the slot data used by the index page.
type IndexSlotData struct {
	Slot      uint64
	Block     bool
	BlockHash string
	Proposer  uint32
	Rewards   int64
}

// IndexData is the data used by the index page.
type IndexData struct {
	Blocks []Block
}

// WaitForConnections waits until beacon app is connected
func (ex *Explorer) WaitForConnections(numConnections int) {
	for {
		if ex.hostNode.PeersConnected() >= numConnections {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// StartExplorer starts the block explorer
func (ex *Explorer) StartExplorer() error {
	err := ex.loadDatabase()
	if err != nil {
		return err
	}

	err = ex.loadP2P()
	if err != nil {
		return err
	}

	err = ex.loadBlockchain()
	if err != nil {
		return err
	}

	ex.syncManager = beacon.NewSyncManager(ex.hostNode, time.Second*5, ex.blockchain)

	ex.syncManager.Start()

	ex.WaitForConnections(1)

	go ex.syncManager.TryInitialSync()

	t := &Template{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}

	e := echo.New()
	e.Renderer = t

	e.GET("/", func(c echo.Context) error {
		blocks := ex.database.GetLatestBlocks(20)

		err := c.Render(http.StatusOK, "index.html", IndexData{blocks})
		if err != nil {
			fmt.Println(err)
		}
		return err
	})

	defer ex.chainDB.Close()

	e.Logger.Fatal(e.Start(":1323"))

	return nil
}
