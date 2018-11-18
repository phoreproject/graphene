package p2p

import (
	"context"
	"fmt"
	"io"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/inconshreveable/log15"
	iaddr "github.com/ipfs/go-ipfs-addr"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-pubsub"

	"github.com/phoreproject/synapse/net"
	"github.com/phoreproject/synapse/pb"
)

// StringToPeerInfo converts a string to a peer info.
func StringToPeerInfo(addrStr string) (*peerstore.PeerInfo, error) {
	addr, err := iaddr.ParseString(addrStr)
	if err != nil {
		return nil, err
	}
	peerinfo, err := peerstore.InfoFromP2pAddr(addr.Multiaddr())
	if err != nil {
		return nil, err
	}
	return peerinfo, nil
}

// RPCServer is a server to manager the P2P module
// RPC server.
type RPCServer struct {
	service        *net.NetworkingService
	subscriptions  map[uint64]*pubsub.Subscription
	subChannels    map[uint64]chan []byte
	cancelChannels map[uint64]chan bool
	currentSubID   *uint64 // this is weird, but all of the methods have to pass the struct in by value
}

// NewRPCServer sets up a server for handling P2P module RPC requests.
func NewRPCServer(netService *net.NetworkingService) RPCServer {
	p := RPCServer{
		service:        netService,
		subscriptions:  make(map[uint64]*pubsub.Subscription),
		subChannels:    make(map[uint64]chan []byte),
		cancelChannels: make(map[uint64]chan bool),
		currentSubID:   new(uint64),
	}
	*p.currentSubID = 0
	return p
}

// GetConnectionStatus gets the status of the P2P connection.
func (p RPCServer) GetConnectionStatus(ctx context.Context, in *empty.Empty) (*pb.ConnectionStatus, error) {
	return &pb.ConnectionStatus{Connected: p.service.IsConnected()}, nil
}

// GetPeers gets the peers for the P2P connection.
func (p RPCServer) GetPeers(ctx context.Context, in *empty.Empty) (*pb.GetPeersResponse, error) {
	peers := p.service.GetPeers()
	peersPb := []*pb.Peer{}
	for _, p := range peers {
		peersPb = append(peersPb, &pb.Peer{Address: p.String()})
	}
	return &pb.GetPeersResponse{Peers: peersPb}, nil
}

// ListenForMessages listens to a subscription and receives
// a stream of messages.
func (p RPCServer) ListenForMessages(in *pb.Subscription, out pb.P2PRPC_ListenForMessagesServer) error {
	if _, success := p.subscriptions[in.ID]; !success {
		return fmt.Errorf("could not find subscription with ID %d", in.ID)
	}

	log15.Debug("listening to new messages on sub", "subID", in.ID)

	for {
		select {
		case msg := <-p.subChannels[in.ID]:
			err := out.Send(&pb.Message{Data: msg})
			if err != nil {
				return err
			}
		case <-p.cancelChannels[in.ID]:
			return io.EOF
		}
	}

}

// Subscribe subscribes to a topic returning a subscription ID.
func (p RPCServer) Subscribe(ctx context.Context, in *pb.SubscriptionRequest) (*pb.Subscription, error) {
	subID := *p.currentSubID
	*p.currentSubID++

	log15.Debug("subscribed to new messages", "topic", in.Topic, "subID", subID)

	p.subChannels[subID] = make(chan []byte)
	p.cancelChannels[subID] = make(chan bool)

	s, err := p.service.RegisterHandler(in.Topic, func(b []byte) error {
		select {
		case p.subChannels[subID] <- b:
		default:
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	p.subscriptions[subID] = s

	return &pb.Subscription{ID: subID}, nil
}

// Unsubscribe unsubscribes from a subscription given a subscription ID.
func (p RPCServer) Unsubscribe(ctx context.Context, in *pb.Subscription) (*empty.Empty, error) {
	if _, success := p.subscriptions[in.ID]; !success {
		return nil, fmt.Errorf("could not find subscription with ID %d", in.ID)
	}

	log15.Debug("unsubscribed to subID", "subID", in.ID)

	// either send it or not. we don't really care if it works.
	// this is dependent on whether the channel is being listened on
	select {
	case p.cancelChannels[in.ID] <- true:
	default:
	}

	close(p.cancelChannels[in.ID])
	close(p.subChannels[in.ID])
	p.subscriptions[in.ID].Cancel()

	return &empty.Empty{}, nil
}

// Broadcast broadcasts a message to a topic.
func (p RPCServer) Broadcast(ctx context.Context, in *pb.MessageAndTopic) (*empty.Empty, error) {
	return &empty.Empty{}, p.service.Broadcast(in.Topic, in.Data)
}

// Connect connects to more peers.
func (p RPCServer) Connect(ctx context.Context, in *pb.Peers) (*pb.ConnectResponse, error) {
	success := true
	for _, peer := range in.Peers {
		pInfo, err := StringToPeerInfo(peer.Address)
		if err != nil {
			return nil, err
		}
		err = p.service.Connect(pInfo)
		if err != nil {
			success = false
			log15.Warn("could not connect to peer", "addr", peer.Address)
			continue
		}
	}
	return &pb.ConnectResponse{Success: success}, nil
}

// GetSettings gets the settings of the P2P connection.
func (p RPCServer) GetSettings(ctx context.Context, in *empty.Empty) (*pb.P2PSettings, error) {
	return &pb.P2PSettings{}, nil
}
