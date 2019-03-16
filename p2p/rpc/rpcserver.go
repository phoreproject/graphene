package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/multiformats/go-multiaddr"
	"io"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
)

// Server is a server to manager the P2P module
// RPC server.
type Server struct {
	hostNode       *p2p.HostNode
	subscriptions  map[uint64]*pubsub.Subscription
	subChannels    map[uint64]chan []byte
	cancelChannels map[uint64]chan bool
	currentSubID   *uint64 // this is weird, but all of the methods have to pass the struct in by value
	lock           *sync.Mutex

	directMessageSubscriptions  map[uint64]uint64
	directMessageSubChannels    map[uint64]chan pb.ListenForDirectMessagesResponse
	directMessageCancelChannels map[uint64]chan bool
}

// NewRPCServer sets up a server for handling P2P module RPC requests.
func NewRPCServer(hostNode *p2p.HostNode) Server {
	p := Server{
		hostNode:                    hostNode,
		subscriptions:               make(map[uint64]*pubsub.Subscription),
		subChannels:                 make(map[uint64]chan []byte),
		cancelChannels:              make(map[uint64]chan bool),
		currentSubID:                new(uint64),
		lock:                        new(sync.Mutex),
		directMessageSubscriptions:  make(map[uint64]uint64),
		directMessageSubChannels:    make(map[uint64]chan pb.ListenForDirectMessagesResponse),
		directMessageCancelChannels: make(map[uint64]chan bool),
	}
	*p.currentSubID = 0
	return p
}

// GetConnectionStatus gets the status of the P2P connection.
func (s Server) GetConnectionStatus(ctx context.Context, in *empty.Empty) (*pb.ConnectionStatus, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return &pb.ConnectionStatus{Connected: s.hostNode.Connected()}, nil
}

// GetPeers gets the peers for the P2P connection.
func (s Server) GetPeers(ctx context.Context, in *empty.Empty) (*pb.GetPeersResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	peers := s.hostNode.GetPeerList()
	peersPb := make([]*pb.Peer, 0)
	for _, p := range peers {
		peersPb = append(peersPb, &pb.Peer{
			PeerID: p.ID.String(),
		})
	}
	return &pb.GetPeersResponse{Peers: peersPb}, nil
}

// ListenForMessages listens to a subscription and receives
// a stream of messages.
func (s Server) ListenForMessages(in *pb.Subscription, out pb.P2PRPC_ListenForMessagesServer) error {
	s.lock.Lock()
	if _, success := s.subscriptions[in.ID]; !success {
		return fmt.Errorf("could not find subscription with ID %d", in.ID)
	}

	messages := s.subChannels[in.ID]
	cancelChan := s.cancelChannels[in.ID]

	s.lock.Unlock()

	for {
		select {
		case msg := <-messages:
			err := out.Send(&pb.Message{Data: msg})
			if err != nil {
				return err
			}
		case <-cancelChan:
			return io.EOF
		}
	}

}

// Subscribe subscribes to a topic returning a subscription ID.
func (s Server) Subscribe(ctx context.Context, in *pb.SubscriptionRequest) (*pb.Subscription, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	subID := *s.currentSubID
	*s.currentSubID++

	subChan := make(chan []byte)
	s.subChannels[subID] = subChan
	s.cancelChannels[subID] = make(chan bool)

	sub, err := s.hostNode.SubscribeMessage(in.Topic, func(data []byte) {
		select {
		case subChan <- data:
		default:
		}
	})

	if err != nil {
		return nil, err
	}

	s.subscriptions[subID] = sub

	return &pb.Subscription{ID: subID}, nil
}

// Unsubscribe unsubscribes from a subscription given a subscription ID.
func (s Server) Unsubscribe(ctx context.Context, in *pb.Subscription) (*empty.Empty, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, success := s.subscriptions[in.ID]; !success {
		return nil, fmt.Errorf("could not find subscription with ID %d", in.ID)
	}

	// either send it or not. we don't really care if it works.
	// this is dependent on whether the channel is being listened on
	select {
	case s.cancelChannels[in.ID] <- true:
	default:
	}

	close(s.cancelChannels[in.ID])
	close(s.subChannels[in.ID])
	s.hostNode.UnsubscribeMessage(s.subscriptions[in.ID])

	delete(s.cancelChannels, in.ID)
	delete(s.subChannels, in.ID)
	delete(s.subscriptions, in.ID)

	return &empty.Empty{}, nil
}

// Broadcast broadcasts a message to a topic.
func (s Server) Broadcast(ctx context.Context, in *pb.MessageAndTopic) (*empty.Empty, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return &empty.Empty{}, s.hostNode.Broadcast(in.Topic, in.Data)
}

// Connect connects to more peers.
func (s Server) Connect(ctx context.Context, in *pb.Peers) (*pb.ConnectResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	success := true
	for _, peerToConnect := range in.Peers {
		peerID, err := peer.IDFromString(peerToConnect.PeerID)
		if err != nil {
			return nil, err
		}
		if peerID == s.hostNode.GetHost().ID() {
			continue
		}
		pInfo := peerstore.PeerInfo{
			ID:    peerID,
			Addrs: []multiaddr.Multiaddr{multiaddr.StringCast(peerToConnect.Address)},
		}
		if err != nil {
			return nil, err
		}
		_, err = s.hostNode.Connect(pInfo)
		if err != nil {
			success = false
			logrus.WithField("addr", peerToConnect.Address).Warn("could not connect to peerToConnect")
			continue
		}
	}
	return &pb.ConnectResponse{Success: success}, nil
}

// GetSettings gets the settings of the P2P connection.
func (s Server) GetSettings(ctx context.Context, in *empty.Empty) (*pb.P2PSettings, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return &pb.P2PSettings{}, nil
}

// SendDirectMessage sends a direct message
func (s Server) SendDirectMessage(ctx context.Context, in *pb.SendDirectMessageRequest) (*empty.Empty, error) {
	id, err := peer.IDFromString(in.PeerID)
	if err != nil {
		return nil, err
	}
	peerToSend, found := s.hostNode.FindPeerByID(id)
	if !found {
		return nil, errors.New("could not find peer")
	}

	message, err := p2p.BytesToMessage(in.GetMessage())
	if err != nil {
		return nil, err
	}

	err = peerToSend.SendMessage(message)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// SubscribeDirectMessage subscribes a direct message
func (s Server) SubscribeDirectMessage(ctx context.Context, req *pb.SubscribeDirectMessageRequest) (*pb.DirectMessageSubscription, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	subID := *s.currentSubID
	*s.currentSubID++

	subChan := make(chan pb.ListenForDirectMessagesResponse)
	s.directMessageSubChannels[subID] = subChan
	s.directMessageCancelChannels[subID] = make(chan bool)

	requestedPeerID := (*peer.ID)(nil)
	if req.PeerID != "" {
		requestedPeerIDDecoded, err := peer.IDFromString(req.PeerID)
		if err != nil {
			return nil, err
		}
		requestedPeerID = &requestedPeerIDDecoded
	}

	sub := s.hostNode.RegisterMessageHandler(req.MessageName, func(sentBy *p2p.Peer, message proto.Message) error {
		if requestedPeerID != nil {
			if sentBy.ID != *requestedPeerID {
				return nil
			}
		}
		data, _ := p2p.MessageToBytes(message)
		m := pb.ListenForDirectMessagesResponse{
			PeerID: sentBy.ID.String(),
			Data:   data,
		}
		select {
		case subChan <- m:
		default:
		}
		return nil
	})

	s.directMessageSubscriptions[subID] = sub

	return &pb.DirectMessageSubscription{
		ID:          subID,
		MessageName: req.MessageName,
	}, nil
}

// UnsubscribeDirectMessage unsubscribes a direct message
func (s Server) UnsubscribeDirectMessage(ctx context.Context, in *pb.DirectMessageSubscription) (*empty.Empty, error) {
	s.hostNode.UnregisterMessageHandler(in.MessageName, in.ID)
	return nil, nil
}

// ListenForDirectMessages listens for direct messages
func (s Server) ListenForDirectMessages(in *pb.DirectMessageSubscription, out pb.P2PRPC_ListenForDirectMessagesServer) error {
	s.lock.Lock()
	if _, success := s.directMessageSubscriptions[in.ID]; !success {
		return fmt.Errorf("could not find subscription with ID %d", in.ID)
	}

	messages := s.directMessageSubChannels[in.ID]
	cancelChan := s.directMessageCancelChannels[in.ID]

	s.lock.Unlock()

	for {
		select {
		case msg := <-messages:
			err := out.Send(&msg)
			if err != nil {
				return err
			}
		case <-cancelChan:
			return io.EOF
		}
	}
}
