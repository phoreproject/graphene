package testcase

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path"
	"time"

	"google.golang.org/grpc/connectivity"

	beaconconfig "github.com/phoreproject/synapse/beacon/config"
	beaconapp "github.com/phoreproject/synapse/beacon/module"

	validatorconfig "github.com/phoreproject/synapse/validator/config"
	validatorapp "github.com/phoreproject/synapse/validator/module"

	shardconfig "github.com/phoreproject/synapse/shard/config"
	shardapp "github.com/phoreproject/synapse/shard/module"

	testframework "github.com/phoreproject/synapse/integrationtests/framework"
	"github.com/phoreproject/synapse/utils"
	"google.golang.org/grpc"
)

// ValidateTest runs a beacon chain and a validator for a few blocks.
type ValidateTest struct {
	beacon     *beaconapp.BeaconApp
	dataDir    string
	validator  *validatorapp.ValidatorApp
	shard      *shardapp.ShardApp
	beaconConn *grpc.ClientConn
	shardConn  *grpc.ClientConn
	err        chan error
}

func (test *ValidateTest) setup() error {
	test.dataDir = path.Join("/tmp", fmt.Sprintf("beacon-%d", time.Now().Unix()))
	err := os.Mkdir(test.dataDir, 0777)
	if err != nil {
		return err
	}

	bApp, err := beaconapp.NewBeaconApp(beaconconfig.Options{
		RPCListen:          "/unix/tmp/beacon.sock",
		ChainCFG:           "regtest.json",
		Resync:             false,
		DataDir:            test.dataDir,
		GenesisTime:        fmt.Sprintf("%d", utils.Now().Unix()),
		InitialConnections: nil,
		P2PListen:          "/ip4/127.0.0.1/tcp/0",
	})
	if err != nil {
		return err
	}

	test.beacon = bApp

	if _, err := os.Stat("/tmp/beacon.sock"); !os.IsNotExist(err) {
		err = os.Remove("/tmp/beacon.sock")
		if err != nil {
			return err
		}
	}

	beaconConn, err := grpc.Dial("unix:///tmp/beacon.sock", grpc.WithInsecure())
	if err != nil {
		return err
	}

	test.beaconConn = beaconConn

	shardConn, err := grpc.Dial("unix:///tmp/shard.sock", grpc.WithInsecure())
	if err != nil {
		return err
	}

	test.shardConn = shardConn

	validatorIndices := make([]string, 256)
	for i := range validatorIndices {
		validatorIndices[i] = fmt.Sprintf("%d", i)
	}

	s, err := shardapp.NewShardApp(shardconfig.Options{
		RPCListen: "/unix/tmp/shard.sock",
		BeaconRPC: "/unix/tmp/beacon.sock",
	})
	if err != nil {
		return err
	}

	test.shard = s

	if _, err := os.Stat("/tmp/shard.sock"); !os.IsNotExist(err) {
		err = os.Remove("/tmp/shard.sock")
		if err != nil {
			return err
		}
	}

	v, err := validatorapp.NewValidatorApp(validatorconfig.Options{
		BeaconRPC:  "/unix/tmp/beacon.sock",
		ShardRPC:   "/unix/tmp/shard.sock",
		Validators: validatorIndices,
		RootKey:    "testnet",
		RPCListen:  "/ip4/127.0.0.1/tcp/0",
		NetworkID:  "regtest",
	})
	if err != nil {
		return err
	}

	test.validator = v

	test.err = make(chan error)

	return nil
}

func (test *ValidateTest) runBeacon() {
	go func() {
		err := test.beacon.Run()
		if err != nil {
			test.err <- err
		}
	}()
}

func (test *ValidateTest) runValidator() {
	t := time.NewTicker(time.Second)
	num := 0
	max := 15
	for test.beaconConn.GetState() != connectivity.Ready {
		if num >= max {
			break
		}
		num++
		<-t.C
	}

	num = 0
	for test.shardConn.GetState() != connectivity.Ready {
		if num >= max {
			break
		}
		num++
		<-t.C
	}

	go func() {
		err := test.validator.Run()
		if err != nil {
			test.err <- err
		}
	}()
}

func (test *ValidateTest) runShard() {
	t := time.NewTicker(time.Second)
	num := 0
	max := 15
	for test.beaconConn.GetState() != connectivity.Ready {
		if num >= max {
			break
		}
		num++
		<-t.C
	}

	go func() {
		err := test.shard.Run()
		if err != nil {
			test.err <- err
		}
	}()
}

func (test *ValidateTest) waitForBlocks() error {
	timer := time.NewTimer(10 * time.Second)

	select {
	case <-timer.C:
		return nil
	case err := <-test.err:
		return err
	}
}

func (test *ValidateTest) exit() {
	test.beacon.Exit()
	test.validator.Exit()
	test.shard.Exit()

	test.beacon.WaitForExit()

	err := os.Remove("/tmp/beacon.sock")
	if err != nil {
		panic(err)
	}

	err = os.Remove("/tmp/shard.sock")
	if err != nil {
		panic(err)
	}

	err = os.RemoveAll(test.dataDir)
	if err != nil {
		panic(err)
	}
}

// Execute implements IntegrationTest
func (test *ValidateTest) Execute(service *testframework.TestService) error {

	err := test.setup()
	if err != nil {
		return errors.Wrap(err, "error setting up test")
	}
	test.runBeacon()
	test.runShard()
	test.runValidator()
	err = test.waitForBlocks()
	if err != nil {
		return errors.Wrap(err, "error running test")
	}

	defer test.exit()

	return nil
}
