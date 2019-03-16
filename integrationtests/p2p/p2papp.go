package p2p

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"

	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/phoreproject/synapse/integrationtests/framework"
	"github.com/phoreproject/synapse/p2p/app"
)

const startPort = 19000
const startRPCPort = 20000
const appCount = 10

// This test ensures that clients can connect and handshake.

// TestCase implements IntegrationTest
type TestCase struct {
	appList []*app.P2PApp
}

// Execute implements IntegrationTest
func (test TestCase) Execute(service *testframework.TestService) error {
	for i := 0; i < appCount; i++ {
		a := test.createApp(i)
		test.appList = append(test.appList, a)
	}

	for _, a := range test.appList {
		err := a.Initialize()
		if err != nil {
			return err
		}
	}

	for _, a := range test.appList {
		go func(appToRun *app.P2PApp) {
			err := appToRun.Run()
			if err != nil {
				panic(err)
			}
		}(a)
	}

	test.connectApps()

	return nil
}

func (test TestCase) createApp(index int) *app.P2PApp {
	config := app.NewConfig()
	config.ListeningAddress = fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", startPort+index)
	config.RPCAddress = fmt.Sprintf("127.0.0.1:%d", startRPCPort+index)
	config.MinPeerCountToWait = 0
	config.HeartBeatInterval = 3 * 1000
	a := app.NewP2PApp(config)

	return a
}

func (test TestCase) connectApps() {
	for _, from := range test.appList {
		for _, to := range test.appList {
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

	for {
		connecting := false
		for _, a := range test.appList {
			for _, peer := range a.GetHostNode().GetPeerList() {
				connecting = connecting || peer.Connecting
			}
		}
		if !connecting {
			break
		}
		logrus.Debug("Waiting for peers to connect...")
		time.Sleep(1 * time.Second)
	}
	logrus.Debug("Peers connected!")
}
