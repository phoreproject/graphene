package main

import (
	"context"
	"fmt"

	"github.com/phoreproject/synapse/net"
	"github.com/phoreproject/synapse/pb"
)

type p2prpcServer struct {
	service *net.NetworkingService
}

func (p *p2prpcServer) GetConnectionStatus(ctx context.Context, in *pb.Empty) (*pb.ConnectionStatus, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *p2prpcServer) GetPeers(ctx context.Context, in *pb.Empty) (*pb.GetPeersResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *p2prpcServer) ListenForMessages(in *pb.SubscriptionRequest, out pb.P2PRPC_ListenForMessagesServer) error {
	handler := func(msg net.Message) error {
		return out.Send(&pb.Message{Response: msg.Data})
	}

	err := p.service.RegisterHandler(in.Topic, handler)
	if err != nil {
		return err
	}

	<-out.Context().Done()

	return p.service.CancelHandler(in.Topic)
}

func (p *p2prpcServer) Connect(ctx context.Context, in *pb.InitialPeers) (*pb.ConnectResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *p2prpcServer) Disconnect(ctx context.Context, in *pb.Empty) (*pb.DisconnectResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *p2prpcServer) GetSettings(ctx context.Context, in *pb.Empty) (*pb.P2PSettings, error) {
	return nil, fmt.Errorf("not implemented")
}
