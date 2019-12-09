package rpc

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/relayer/mempool"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

type server struct {
	mempools map[uint64]*mempool.ShardMempool
}

// GetListeningAddresses gets the listening addresses of the relayer P2P protocol.
func (server) GetListeningAddresses(context.Context, *empty.Empty) (*pb.ListeningAddressesResponse, error) {
	panic("implement me")
}

// Connect connects P2P to a certain node.
func (server) Connect(context.Context, *pb.ConnectMessage) (*empty.Empty, error) {
	panic("implement me")
}

// SubmitTransaction submits a transaction to the relayer.
func (s server) SubmitTransaction(ctx context.Context, tx *pb.SubmitTransactionRequest) (*empty.Empty, error) {
	if mem, found := s.mempools[tx.ShardID]; found {
		err := mem.Add(tx.Transaction.TransactionData)
		return &empty.Empty{}, err
	}
	return nil, fmt.Errorf("not tracking shard %d", tx.ShardID)
}

// Serve serves the RPC server
func Serve(proto string, listenAddr string, mempools map[uint64]*mempool.ShardMempool) error {
	logrus.WithField("listen", listenAddr).Info("relayer listening for RPC")
	lis, err := net.Listen(proto, listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterRelayerRPCServer(s, &server{
		mempools: mempools,
	})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
