package testcase

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/rpc"

	crypto "github.com/libp2p/go-libp2p-crypto"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	logger "github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/integrationtests/framework"
	"github.com/phoreproject/synapse/p2p"
)

// P2pTest implements IntegrationTest
type P2pTest struct {
}

type testNode struct {
	*rpc.HostNode
	nodeID int
}

// Execute implements IntegrationTest
func (test P2pTest) Execute(service *testframework.TestService) error {
	logger.SetLevel(logger.TraceLevel)

	hostNode0, err := createHostNode(0)
	if err != nil {
		logger.Warn(err)
		return err
	}
	hostNode1, err := createHostNode(1)
	if err != nil {
		logger.Warn(err)
		return err
	}

	connectToPeer(hostNode0, hostNode1)
	connectToPeer(hostNode1, hostNode0)

	peer1 := hostNode0.GetPeerList()[0].(p2p.LocalPeerNode)

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
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 19000+index))
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

	database := db.NewInMemoryDB()
	c := config.MainNetConfig
	validators := []beacon.InitialValidatorEntry{}
	//randaoCommitment := chainhash.HashH([]byte("test"))
	for i := 0; i <= int(c.EpochLength)*(c.TargetCommitteeSize*2); i++ {
		validators = append(validators, beacon.InitialValidatorEntry{
			//PubKey:                bls.PublicKey{},
			//ProofOfPossession:     bls.Signature{},
			WithdrawalShard:       1,
			WithdrawalCredentials: chainhash.Hash{},
			//RandaoCommitment:      randaoCommitment,
		})
	}
	blockchain, err := beacon.NewBlockchainWithInitialValidators(database, &c, validators, true)

	hostNode, err := rpc.NewHostNode(createNodeAddress(index), publicKey, privateKey, p2p.NewMainRPCServer(blockchain), p2p.LocalPeerNodeCreator{})
	if err != nil {
		logger.WithField("Function", "createHostNode").Warn(err)
		return nil, err
	}

	node := &testNode{
		HostNode: hostNode,
		nodeID:      index,
	}

	node.Run()

	return node, nil
}

func connectToPeer(hostNode *testNode, target *testNode) rpc.RpcPeerNode {
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
