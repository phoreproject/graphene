package p2p

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/libp2p/go-libp2p-peerstore"
	beaconapp "github.com/phoreproject/synapse/beacon/app"
	"github.com/phoreproject/synapse/integrationtests/framework"
	p2papp "github.com/phoreproject/synapse/p2p/app"
)

// TestCaseP2PBeacon implements IntegrationTest
type TestCaseP2PBeacon struct {
	p2pAppList []*p2papp.P2PApp
	beacon     *beaconapp.BeaconApp
}

// Execute implements IntegrationTest
func (test TestCaseP2PBeacon) Execute(service *testframework.TestService) error {
	for i := 0; i < 10; i++ {
		app := test.createP2PApp(i)
		test.p2pAppList = append(test.p2pAppList, app)
	}

	for _, a := range test.p2pAppList {
		err := a.Initialize()
		if err != nil {
			return err
		}
	}

	for _, a := range test.p2pAppList {
		go func(appToRun *p2papp.P2PApp) {
			err := appToRun.Run()
			if err != nil {
				panic(err)
			}
		}(a)
	}

	tempDataDir := path.Join(os.TempDir(), fmt.Sprintf("beacon-%d", time.Now().Unix()))

	err := os.MkdirAll(tempDataDir, 0777)
	if err != nil {
		return err
	}

	test.connectApps()

	errChan := make(chan error)

	test.beacon = test.createBeaconApp(tempDataDir)
	go func() {
		err = test.beacon.Run()
		if err != nil {
			errChan <- err
		}
	}()

	t := time.NewTimer(1 * time.Second)
	select {
	case <-t.C:
		test.beacon.Exit()
	case <-errChan:
		return err
	}

	return os.RemoveAll(tempDataDir)
}

func (test TestCaseP2PBeacon) createP2PApp(index int) *p2papp.P2PApp {
	config := p2papp.NewConfig()
	config.ListeningAddress = fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 29000+index)
	config.RPCAddress = fmt.Sprintf("127.0.0.1:%d", 30000+index)
	config.MinPeerCountToWait = 0
	config.HeartBeatInterval = 3 * 1000
	app := p2papp.NewP2PApp(config)

	return app
}

func (test TestCaseP2PBeacon) createBeaconApp(dataDir string) *beaconapp.BeaconApp {
	config := beaconapp.NewConfig()
	config.P2PAddress = "127.0.0.1:30000"
	config.RPCAddress = "127.0.0.1:35000"
	config.DataDirectory = dataDir
	config.IsIntegrationTest = true
	app := beaconapp.NewBeaconApp(config)
	return app
}

func (test TestCaseP2PBeacon) connectApps() {
	for _, from := range test.p2pAppList {
		for _, to := range test.p2pAppList {
			if from == to {
				continue
			}
			_, err := from.GetHostNode().Connect(peerstore.PeerInfo{
				ID:    to.GetHostNode().GetHost().ID(),
				Addrs: to.GetHostNode().GetHost().Addrs(),
			})
			if err != nil {
				panic(err)
			}
		}
	}
}
