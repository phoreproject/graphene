package testcase

import (
	"fmt"
	"os"
	"path"
	"time"

	"google.golang.org/grpc/connectivity"

	beaconapp "github.com/phoreproject/synapse/beacon/app"

	"github.com/phoreproject/synapse/beacon/config"

	testframework "github.com/phoreproject/synapse/integrationtests/framework"
	"github.com/phoreproject/synapse/utils"
	validatorapp "github.com/phoreproject/synapse/validator/app"
	"google.golang.org/grpc"
)

// ValidateTest runs a beacon chain and a validator for a few blocks.
type ValidateTest struct {
	beacon     *beaconapp.BeaconApp
	dataDir    string
	validator  *validatorapp.ValidatorApp
	beaconConn *grpc.ClientConn
	err        chan error
}

func (test *ValidateTest) setup() error {
	test.dataDir = path.Join("/tmp", fmt.Sprintf("beacon-%d", time.Now().Unix()))
	err := os.Mkdir(test.dataDir, 0777)
	if err != nil {
		return err
	}

	f, err := os.Open("regtest.json")
	if err != nil {
		panic(err)
	}

	beaconConfig, err := beaconapp.ReadChainFileToConfig(f)
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	beaconConfig.RPCProto = "unix"
	beaconConfig.RPCAddress = "/tmp/beacon.sock"
	beaconConfig.GenesisTime = uint64(utils.Now().Unix())
	beaconConfig.Resync = true
	beaconConfig.DataDirectory = test.dataDir
	beaconConfig.NetworkConfig = &config.LocalnetConfig

	test.beacon = beaconapp.NewBeaconApp(*beaconConfig)

	beaconConn, err := grpc.Dial("unix:///tmp/beacon.sock", grpc.WithInsecure())
	if err != nil {
		return err
	}

	test.beaconConn = beaconConn

	validatorIndices := make([]uint32, 256)
	for i := range validatorIndices {
		validatorIndices[i] = uint32(i)
	}

	validatorConfig := validatorapp.ValidatorConfig{
		BlockchainConn:   beaconConn,
		ValidatorIndices: validatorIndices,
		RootKey:          "testnet",
		NetworkConfig:    &config.LocalnetConfig,
	}

	test.validator = validatorapp.NewValidatorApp(validatorConfig)

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

	go func() {
		err := test.validator.Run()
		if err != nil {
			test.err <- err
		}
	}()
}

const numBlocks = 100

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

	err := os.Remove("/tmp/beacon.sock")
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
		return err
	}
	test.runBeacon()
	test.runValidator()
	err = test.waitForBlocks()
	if err != nil {
		return err
	}

	defer test.exit()

	return nil
}
