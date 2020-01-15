package explorer

import (
	"encoding/json"
	"flag"
	"os"
	"strings"

	beaconModule "github.com/phoreproject/synapse/beacon/module"
	"github.com/phoreproject/synapse/p2p"
	shardConfig "github.com/phoreproject/synapse/shard/config"
)

// Config is the explorer config
type Config struct {
	ConfigFileName string `json:"config,omitempty"`
	DbDriver       string `json:"dbdriver,omitempty"`
	DbHost         string `json:"dbhost,omitempty"`
	DbDatabase     string `json:"dbdatabase,omitempty"`
	DbUser         string `json:"dbuser,omitempty"`
	DbPassword     string `json:"dbpassword,omitempty"`

	ChainConfig string `json:"chainconfig,omitempty"`
	Resync      bool   `json:"resync,omitempty"`
	DataDir     string `json:"datadir,omitempty"`
	Connect     string `json:"connect,omitempty"`
	Listen      string `json:"listen,omitempty"`
	RPCListen   string `json:"rpclisten,omitempty"`

	ShardListen string `json:"shardlisten,omitempty"`

	Level string `json:"level,omitempty"`

	beaconConfig *beaconModule.Config
	shardConfig  *shardConfig.Options
}

func newConfig() *Config {
	return &Config{}
}

func loadConfigFromFile(fileName string, config *Config) error {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return err
	}

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	return d.Decode(&config)
}

func loadConfigFromCommandLine(config *Config) {
	configFile := flag.String("config", "synapseexplorer.cfg", "Config file name")
	dbdriver := flag.String("dbdriver", "sqlite", "Database driver, the value can be sqlite, mysql")
	dbhost := flag.String("dbhost", "synapseexplorer.sqlite", "Database host")
	dbdatabase := flag.String("dbdatabase", "synapseexplorer.sqlite", "Database name")
	dbuser := flag.String("dbuser", "synapseexplorer.sqlite", "Database user name")
	dbpassword := flag.String("dbpassword", "synapseexplorer.sqlite", "Database password")

	chainconfig := flag.String("chainconfig", "testnet.json", "file of chain config")
	resync := flag.Bool("resync", false, "resyncs the blockchain if this is set")
	datadir := flag.String("datadir", "", "location to store blockchain data")
	connect := flag.String("connect", "", "comma separated multiaddrs")
	listen := flag.String("listen", "/ip4/0.0.0.0/tcp/11781", "specifies the address to listen on")
	rpcListen := flag.String("rpclisten", "/ip4/0.0.0.0/tcp/11781", "specifies the address to RPC listen on")
	shardListen := flag.String("shardlisten", "/ip4/0.0.0.0/tcp/11781", "specifies the address to listen on")

	level := flag.String("level", "info", "log level")
	flag.Parse()

	config.ConfigFileName = *configFile
	config.DbDriver = *dbdriver
	config.DbHost = *dbhost
	config.DbDatabase = *dbdatabase
	config.DbUser = *dbuser
	config.DbPassword = *dbpassword

	config.ChainConfig = *chainconfig
	config.Resync = *resync
	config.DataDir = *datadir
	config.Connect = *connect
	config.Listen = *listen
	config.RPCListen = *rpcListen
	config.Level = *level

	config.ShardListen = *shardListen
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func mergeConfigFromConfigFile(config *Config) {
	if config.ConfigFileName == "" {
		return
	}

	configFromFile := newConfig()
	err := loadConfigFromFile(config.ConfigFileName, configFromFile)
	if err != nil {
		return
	}

	if !isFlagPassed("dbdriver") {
		config.DbDriver = configFromFile.DbDriver
	}
	if !isFlagPassed("dbhost") {
		config.DbHost = configFromFile.DbHost
	}
	if !isFlagPassed("dbdatabase") {
		config.DbDatabase = configFromFile.DbDatabase
	}
	if !isFlagPassed("dbuser") {
		config.DbUser = configFromFile.DbUser
	}
	if !isFlagPassed("dbpassword") {
		config.DbPassword = configFromFile.DbPassword
	}
	if !isFlagPassed("chainconfig") {
		config.ChainConfig = configFromFile.ChainConfig
	}
	if !isFlagPassed("resync") {
		config.Resync = configFromFile.Resync
	}
	if !isFlagPassed("datadir") {
		config.DataDir = configFromFile.DataDir
	}
	if !isFlagPassed("connect") {
		config.Connect = configFromFile.Connect
	}
	if !isFlagPassed("listen") {
		config.Listen = configFromFile.Listen
	}
	if !isFlagPassed("level") {
		config.Level = configFromFile.Level
	}
}

// LoadConfig loads the config
func LoadConfig() *Config {
	config := newConfig()

	loadConfigFromCommandLine(config)
	mergeConfigFromConfigFile(config)

	prepareConfig(config)

	return config
}

func prepareConfig(config *Config) {
	f, err := os.Open(config.ChainConfig)
	if err != nil {
		panic(err)
	}

	beaconConfig, err := beaconModule.ReadChainFileToConfig(f)
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	beaconConfig.DataDirectory = config.DataDir
	beaconConfig.Resync = config.Resync
	beaconConfig.ListeningAddress = config.Listen

	initialPeers, err := p2p.ParseInitialConnections(strings.Split(config.Connect, ","))
	if err != nil {
		panic(err)
	}
	beaconConfig.DiscoveryOptions.PeerAddresses = append(beaconConfig.DiscoveryOptions.PeerAddresses, initialPeers...)

	config.beaconConfig = beaconConfig

	config.shardConfig = &shardConfig.Options{
		RPCListen: config.ShardListen,
		BeaconRPC: config.RPCListen,
	}
}
