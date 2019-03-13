package p2p

import (
	"fmt"
	"math/rand"
	"time"

	peerstore "github.com/libp2p/go-libp2p-peerstore"
	beaconapp "github.com/phoreproject/synapse/beacon/app"
	testframework "github.com/phoreproject/synapse/integrationtests/framework"
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

	for _, app := range test.p2pAppList {
		go func() {
			err := app.Run()
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	}

	// Wait until host nodes are created
	time.Sleep(300 * time.Millisecond)

	test.connectApps()

	test.beacon = test.createBeaconApp()
	test.beacon.Run()

	time.Sleep(5 * time.Second)

	return nil
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

func (test TestCaseP2PBeacon) createBeaconApp() *beaconapp.BeaconApp {
	config := beaconapp.NewConfig()
	config.P2PAddress = "127.0.0.1:30000"
	config.RPCAddress = "127.0.0.1:35000"
	app := beaconapp.NewBeaconApp(config)
	return app
}

func (test TestCaseP2PBeacon) connectApps() {
	for i := 0; i < len(test.p2pAppList); i++ {
		for k := 0; k < peerCount; k++ {
			peerIndex := rand.Int() % len(test.p2pAppList)
			if i == 1 && k == 0 {
				peerIndex = 0
			}
			test.p2pAppList[i].GetHostNode().Connect(&peerstore.PeerInfo{
				ID:    test.p2pAppList[peerIndex].GetHostNode().GetHost().ID(),
				Addrs: test.p2pAppList[peerIndex].GetHostNode().GetHost().Addrs(),
			})
		}
	}
}
