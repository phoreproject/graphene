package testcase

import (
	"crypto/rand"
	"fmt"

	crypto "github.com/libp2p/go-libp2p-crypto"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	logger "github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/integrationtests/framework"
	p2p "github.com/phoreproject/synapse/p2p"
)

// P2pTest implements IntegrationTest
type P2pTest struct {
}

// Execute implements IntegrationTest
func (test P2pTest) Execute(service *testframework.TestService) error {
	logger.SetLevel(logger.TraceLevel)

	hostNode0, err := createHostNode(0)
	if err != nil {
		logger.Warn(err)
	}
	hostNode1, err := createHostNode(1)
	if err != nil {
		logger.Warn(err)
	}

	connectToPeer(hostNode0, hostNode1)
	connectToPeer(hostNode1, hostNode0)

	select {}

	return nil
}

func createNodeAddress(index int) ma.Multiaddr {
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 9000+index))
	if err != nil {
		logger.WithField("Function", "createNodeAddress").Warn(err)
		return nil
	}
	return addr
}

func createHostNode(index int) (*p2p.HostNode, error) {
	privateKey, publicKey, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		logger.WithField("Function", "createHostNode").Warn(err)
		return nil, err
	}

	hostNode, err := p2p.NewHostNode(createNodeAddress(index), publicKey, privateKey)
	if err != nil {
		logger.WithField("Function", "createHostNode").Warn(err)
		return nil, err
	}

	hostNode.Run()

	return hostNode, nil
}

func connectToPeer(hostNode *p2p.HostNode, target *p2p.HostNode) *p2p.PeerNode {
	addrs := target.GetHost().Addrs()

	peerInfo := peerstore.PeerInfo{
		ID:    target.GetHost().ID(),
		Addrs: addrs,
	}

	logger.WithField("Function", "connectToPeer").Warn(peerInfo.ID.Pretty())

	node, err := hostNode.Connect(&peerInfo)
	if err != nil {
		logger.WithField("Function", "connectToPeer").Warn(err)
		return nil
	}
	return node
}
