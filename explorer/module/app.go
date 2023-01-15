package module

import (
	"context"
	"fmt"
	"os"

	"github.com/jinzhu/gorm"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/chainfile"
	"github.com/phoreproject/synapse/beacon/config"
	db "github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/explorer"
	beaconconfig "github.com/phoreproject/synapse/explorer/config"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	_ "github.com/mattn/go-sqlite3"
)

const explorerVersion = "0.0.1"

// ExplorerApp runs the shard microservice which handles state execution and fork choice on shards.
type ExplorerApp struct {
	beaconRPC pb.BlockchainRPCClient
	explorer  *explorer.Explorer
	db        *gorm.DB
}

// NewExplorerApp creates a new shard app given a config.
func NewExplorerApp(options beaconconfig.Options) (*ExplorerApp, error) {
	beaconAddr, err := utils.MultiaddrStringToDialString(options.BeaconRPC)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing beacon RPC address")
	}

	cc, err := grpc.Dial(beaconAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("could not connect to beacon host at %s with error: %s", options.BeaconRPC, err)
	}

	f, err := os.Open(options.ChainCFG)
	if err != nil {
		logrus.Fatal(err)
	}

	chainConfig, err := chainfile.ReadChainFile(f)
	if err != nil {
		logrus.Fatal(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	networkConfig, found := config.NetworkIDs[chainConfig.NetworkID]
	if !found {
		return nil, fmt.Errorf("error getting network config for ID: %s", chainConfig.NetworkID)
	}

	explorerConfig := &explorer.Config{
		DataDirectory: options.DataDirectory,
		NetworkConfig: &networkConfig,
	}

	explorerDb, err := gorm.Open("sqlite3", "/tmp/gorm.db")
	if err != nil {
		return nil, err
	}

	beaconRPC := pb.NewBlockchainRPCClient(cc)

	genesisTimeRes, err := beaconRPC.GetGenesisTime(context.TODO(), &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	validators, err := chainConfig.GetInitialValidators()
	if err != nil {
		return nil, err
	}

	genesisTime := genesisTimeRes.GenesisTime

	blockchain, err := beacon.NewBlockchainWithInitialValidators(db.NewInMemoryDB(), &networkConfig, validators, false, genesisTime)
	if err != nil {
		return nil, err
	}

	ex, err := explorer.NewExplorer(*explorerConfig, explorerDb, beaconRPC, blockchain)
	if err != nil {
		return nil, err
	}

	a := &ExplorerApp{beaconRPC: pb.NewBlockchainRPCClient(cc), explorer: ex, db: explorerDb}

	return a, nil
}

// Run runs the shard app.
func (s *ExplorerApp) Run() error {
	logrus.Infof("initializing block explorer v%s", explorerVersion)

	return s.explorer.StartExplorer()
}

// Exit exits the module.
func (s *ExplorerApp) Exit() {
}
