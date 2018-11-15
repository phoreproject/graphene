package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/phoreproject/synapse/pb"

	logger "github.com/inconshreveable/log15"
	iaddr "github.com/ipfs/go-ipfs-addr"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
	network "github.com/phoreproject/synapse/net"
)

func parseInitialConnections(in string) ([]*peerstore.PeerInfo, error) {
	currentAddr := ""

	peers := []*peerstore.PeerInfo{}

	for i := range in {
		if in[i] == ',' {
			addr, err := iaddr.ParseString(currentAddr)
			currentAddr = ""
			if err != nil {
				return nil, err
			}
			peerinfo, err := pstore.InfoFromP2pAddr(addr.Multiaddr())
			if err != nil {
				return nil, err
			}

			peers = append(peers, peerinfo)
		}
		currentAddr = currentAddr + string(in[i])
	}

	return peers, nil
}

func main() {
	listen := flag.String("listen", "/ip4/0.0.0.0/tcp/11781", "specifies the address to listen on")
	initialConnections := flag.String("connect", "", "comma separated multiaddrs")
	rpcConnect := flag.String("rpclisten", "127.0.0.1:11783", "host and port for RPC server to listen on")
	flag.Parse()

	logger.Debug("starting p2p service")

	logger.Info("initializing net")
	ps, err := parseInitialConnections(*initialConnections)
	if err != nil {
		panic(err)
	}

	sourceMultiAddr, err := multiaddr.NewMultiaddr(*listen)
	if err != nil {
		fmt.Printf("address %s is invalid", *listen)
		return
	}

	priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		panic(err)
	}

	network, err := network.NewNetworkingService(&sourceMultiAddr, priv)
	if err != nil {
		panic(err)
	}

	logger.Info("connecting to bootnodes")

	for _, p := range ps {
		err = network.Connect(p)
		if err != nil {
			panic(err)
		}
	}
	if err != nil {
		panic(err)
	}

	logger.Info("starting P2P RPC service")
	lis, err := net.Listen("tcp", *rpcConnect)
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	pb.RegisterP2PRPCServer(s, &p2prpcServer{service: &network})
	reflection.Register(s)
	err = s.Serve(lis)
	if err != nil {
		panic(err)
	}
}
