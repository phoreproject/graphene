package rpc

import (
	"context"
	"fmt"
	"net"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/relayer/shardrelayer"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	relayers map[uint64]*shardrelayer.ShardRelayer
	*pb.UnimplementedRelayerRPCServer
}

// GetStateKey gets a key from the state for a certain shard.
func (s *server) GetStateKey(ctx context.Context, req *pb.GetStateKeyRequest) (*pb.StateKey, error) {
	if r, found := s.relayers[req.ShardID]; found {
		var keyH chainhash.Hash
		copy(keyH[:], req.Key)
		val, err := r.GetStateKey(keyH)
		if err != nil {
			return nil, err
		}
		return &pb.StateKey{
			Value: val[:],
		}, nil
	}
	return nil, fmt.Errorf("not tracking shard %d", req.ShardID)
}

// GetStateKeys get multiple keys from the state for a certain shard.
func (s *server) GetStateKeys(ctx context.Context, req *pb.GetStateKeysRequest) (*pb.StateKeys, error) {
	if r, found := s.relayers[req.ShardID]; found {
		keyH, err := chainhash.BytesToHashes(req.Key)
		if err != nil {
			return nil, err
		}

		vals, err := r.GetStateKeys(keyH)
		if err != nil {
			return nil, err
		}

		return &pb.StateKeys{
			Values: chainhash.HashesToBytes(vals),
		}, nil
	}
	return nil, fmt.Errorf("not tracking shard %d", req.ShardID)
}

// GetListeningAddresses gets the listening addresses of the relayer P2P protocol.
func (*server) GetListeningAddresses(context.Context, *emptypb.Empty) (*pb.ListeningAddressesResponse, error) {
	panic("implement me")
}

// Connect connects P2P to a certain node.
func (*server) Connect(context.Context, *pb.ConnectMessage) (*emptypb.Empty, error) {
	panic("implement me")
}

// SubmitTransaction submits a transaction to the relayer.
func (s *server) SubmitTransaction(ctx context.Context, tx *pb.SubmitTransactionRequest) (*emptypb.Empty, error) {
	if r, found := s.relayers[tx.ShardID]; found {
		err := r.GetMempool().Add(tx.Transaction.TransactionData)
		return &emptypb.Empty{}, err
	}
	return nil, fmt.Errorf("not tracking shard %d", tx.ShardID)
}

// Serve serves the RPC server
func Serve(proto string, listenAddr string, relayers map[uint64]*shardrelayer.ShardRelayer) error {
	logrus.WithField("listen", listenAddr).Info("relayer listening for RPC")
	lis, err := net.Listen(proto, listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterRelayerRPCServer(s, &server{
		relayers: relayers,
	})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
