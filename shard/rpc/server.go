package rpc

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/shard/chain"
	"github.com/prysmaticlabs/go-ssz"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

// ShardRPCServer handles incoming commands for the shard module.
type ShardRPCServer struct {
	sm *chain.ShardMux
}

// GetStateKey gets a key from the current state.
func (s *ShardRPCServer) GetStateKey(ctx context.Context, req *pb.GetStateKeyRequest) (*pb.GetStateKeyResponse, error) {
	manager, err := s.sm.GetManager(req.ShardID)
	if err != nil {
		return nil, err
	}

	keyH, err := chainhash.NewHash(req.Key)
	if err != nil {
		return nil, err
	}

	val, err := manager.GetKey(*keyH)
	if err != nil {
		return nil, err
	}

	return &pb.GetStateKeyResponse{
		Value: val[:],
	}, nil
}

// SubmitTransaction submits a transaction to a certain shard.
func (s *ShardRPCServer) SubmitTransaction(ctx context.Context, tx *pb.ShardTransactionSubmission) (*empty.Empty, error) {
	manager, err := s.sm.GetManager(tx.ShardID)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, manager.SubmitTransaction(tx.Transaction.TransactionData)
}

// SubscribeToShard instructs the shard module to subscribe to a specific shard and start downloading blocks. This is a
// no-op until we implement the P2P network.
func (s *ShardRPCServer) SubscribeToShard(ctx context.Context, req *pb.ShardSubscribeRequest) (*empty.Empty, error) {
	blockHash, err := chainhash.NewHash(req.BlockHash)
	if err != nil {
		return nil, err
	}

	if !s.sm.IsManaging(req.ShardID) {
		s.sm.StartManaging(req.ShardID, chain.ShardChainInitializationParameters{
			RootBlockHash: *blockHash,
			RootSlot:      req.CrosslinkSlot,
		})
	}

	return &empty.Empty{}, nil
}

// UnsubscribeFromShard instructs the shard module to unsubscribe from a specific shard including disconnecting from
// unnecessary peers and clearing the database of any unnecessary state. No-op until we implement the P2P network.
func (s *ShardRPCServer) UnsubscribeFromShard(ctx context.Context, req *pb.ShardUnsubscribeRequest) (*empty.Empty, error) {
	s.sm.StopManaging(req.ShardID)

	return &empty.Empty{}, nil
}

// GetBlockHashAtSlot gets the block hash at a specific slot on a specific shard.
func (s *ShardRPCServer) GetBlockHashAtSlot(ctx context.Context, req *pb.SlotRequest) (*pb.BlockHashResponse, error) {
	if req.Slot == 0 {
		genesisBlock := primitives.GetGenesisBlockForShard(req.Shard)

		genesisHash, err := ssz.HashTreeRoot(genesisBlock)
		if err != nil {
			return nil, err
		}

		return &pb.BlockHashResponse{
			BlockHash: genesisHash[:],
		}, nil
	}

	manager, err := s.sm.GetManager(req.Shard)
	if err != nil {
		return nil, err
	}

	blockNode, err := manager.Chain.GetNodeBySlot(req.Slot)
	if err != nil {
		return nil, err
	}

	return &pb.BlockHashResponse{
		BlockHash: blockNode.BlockHash[:],
	}, nil
}

// GenerateBlockTemplate generates a block template using transactions and witnesses for a certain shard at a certain
// slot.
func (s *ShardRPCServer) GenerateBlockTemplate(ctx context.Context, req *pb.BlockGenerationRequest) (*pb.ShardBlock, error) {
	finalizedBeaconHash, err := chainhash.NewHash(req.FinalizedBeaconHash)
	if err != nil {
		return nil, err
	}

	manager, err := s.sm.GetManager(req.Shard)
	if err != nil {
		return nil, err
	}

	tipNode, err := manager.Chain.Tip()
	if err != nil {
		return nil, err
	}

	transactionsToInclude, newStateRoot, err := manager.Mempool.GetTransactions(-1)
	if err != nil {
		return nil, err
	}

	transactions := make([]primitives.ShardTransaction, len(transactionsToInclude))
	for i := range transactionsToInclude {
		transactions[i] = primitives.ShardTransaction{
			TransactionData: transactionsToInclude[i],
		}
	}

	transactionRoot, _ := ssz.HashTreeRoot(transactions)

	// for now, block is empty, but we'll fill this in eventually
	block := &primitives.ShardBlock{
		Header: primitives.ShardBlockHeader{
			PreviousBlockHash:   tipNode.BlockHash,
			Slot:                req.Slot,
			Signature:           [48]byte{},
			StateRoot:           *newStateRoot,
			TransactionRoot:     transactionRoot,
			FinalizedBeaconHash: *finalizedBeaconHash,
		},
		Body: primitives.ShardBlockBody{
			Transactions: transactions,
		},
	}

	return block.ToProto(), nil
}

// SubmitBlock submits a block to the shard chain.
func (s *ShardRPCServer) SubmitBlock(ctx context.Context, req *pb.ShardBlockSubmission) (*empty.Empty, error) {
	block, err := primitives.ShardBlockFromProto(req.Block)
	if err != nil {
		return nil, err
	}

	manager, err := s.sm.GetManager(req.Shard)
	if err != nil {
		return nil, err
	}

	err = manager.SubmitBlock(*block)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

var _ pb.ShardRPCServer = &ShardRPCServer{}

// Serve serves the RPC server
func Serve(proto string, listenAddr string, mux *chain.ShardMux) error {
	lis, err := net.Listen(proto, listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterShardRPCServer(s, &ShardRPCServer{mux})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
