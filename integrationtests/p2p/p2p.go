package testcase

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/phoreproject/synapse/pb"

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

type testNode struct {
	*p2p.HostNode
	nodeID int
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

	peer1 := hostNode0.GetPeerList()[0]

	for i := 0; i < 10; i++ {
		message := fmt.Sprintf("Test message of %d", i)
		fmt.Printf("Request: %s\n", message)
		response, _ := peer1.GetClient().Test(hostNode0.GetContext(), &pb.TestMessage{Message: message})
		fmt.Printf("Response: %s\n", response)

		time.Sleep(500 * time.Millisecond)
	}

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

func createHostNode(index int) (*testNode, error) {
	privateKey, publicKey, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		logger.WithField("Function", "createHostNode").Warn(err)
		return nil, err
	}

	hostNode, err := p2p.NewHostNode(createNodeAddress(index), publicKey, privateKey, p2p.NewMainRPCServer())
	if err != nil {
		logger.WithField("Function", "createHostNode").Warn(err)
		return nil, err
	}

	node := &testNode{
		HostNode: hostNode,
		nodeID:   index,
	}

	node.Run()

	return node, nil
}

func connectToPeer(hostNode *testNode, target *testNode) *p2p.PeerNode {
	addrs := target.GetHost().Addrs()

	peerInfo := peerstore.PeerInfo{
		ID:    target.GetHost().ID(),
		Addrs: addrs,
	}

	logger.WithField("Function", "connectToPeer").Trace(peerInfo.ID.Pretty())

	node, err := hostNode.Connect(&peerInfo)
	if err != nil {
		logger.WithField("Function", "connectToPeer").Warn(err)
		return nil
	}
	return node
}
