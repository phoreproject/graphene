package integrationtests

import (
	"crypto/rand"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"

	crypto "github.com/libp2p/go-libp2p-crypto"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
	network "github.com/phoreproject/synapse/net"
	logger "github.com/sirupsen/logrus"
)

// SynapseP2pTest implements IntegrationTest
type SynapseP2pTest struct {
}

func parseInitialConnections(in string) ([]*peerstore.PeerInfo, error) {
	currentAddr := ""

	peers := []*peerstore.PeerInfo{}

	for i := range in {
		if in[i] == ',' {
			peerinfo, err := p2p.StringToPeerInfo(currentAddr)
			if err != nil {
				return nil, err
			}
			currentAddr = ""

			peers = append(peers, peerinfo)
		}
		currentAddr = currentAddr + string(in[i])
	}

	return peers, nil
}

// Execute implements IntegrationTest
func (test SynapseP2pTest) Execute(service *TestService) error {
	listen := service.GetArgString("listen", "/ip4/0.0.0.0/tcp/11781")
	initialConnections := service.GetArgString("connect", "")
	rpcConnect := service.GetArgString("rpclisten", "127.0.0.1:11783")

	logger.Debug("starting p2p service")

	logger.Info("initializing net")
	ps, err := parseInitialConnections(initialConnections)
	if err != nil {
		return err
	}

	sourceMultiAddr, err := multiaddr.NewMultiaddr(listen)
	if err != nil {
		logger.WithField("addr", listen).Fatal("address is invalid")
		return err
	}

	priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return err
	}

	network, err := network.NewNetworkingService(&sourceMultiAddr, priv)
	if err != nil {
		return err
	}

	logger.Info("connecting to bootnodes")

	for _, p := range ps {
		err = network.Connect(p)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}

	logger.Info("starting P2P RPC service")
	lis, err := net.Listen("tcp", rpcConnect)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	pb.RegisterP2PRPCServer(s, p2p.NewRPCServer(&network))
	reflection.Register(s)
	err = s.Serve(lis)
	if err != nil {
		return err
	}

	return nil
}
