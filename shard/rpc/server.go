package rpc

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/p2p"
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
	hn *p2p.HostNode
	config *config.Config
}

// GetListeningAddresses gets listening addresses for the shard module.
func (s *ShardRPCServer) GetListeningAddresses(context.Context, *empty.Empty) (*pb.ListeningAddressesResponse, error) {
	addrs := s.hn.GetHost().Addrs()

	info := peer.AddrInfo{
		ID: s.hn.GetHost().ID(),
		Addrs: addrs,
	}

	p2paddrs, err := peer.AddrInfoToP2pAddrs(&info)
	if err != nil {
		return nil, err
	}

	addrStrings := make([]string, len(p2paddrs))
	for i := range p2paddrs {
		addrStrings[i] = p2paddrs[i].String()
	}

	return &pb.ListeningAddressesResponse{
		Addresses:            addrStrings,
	}, nil
}

// Connect connects to a peer
func (s *ShardRPCServer) Connect(ctx context.Context, connectMsg *pb.ConnectMessage) (*empty.Empty, error) {
	addr := connectMsg.Address

	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}

	pi, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return nil, err
	}

	err = s.hn.Connect(ctx, *pi)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

// GetSlotNumber gets the current tip slot and hash.
func (s *ShardRPCServer) GetSlotNumber(ctx context.Context, req *pb.SlotNumberRequest) (*pb.SlotNumberResponse, error) {
	mgr, err := s.sm.GetManager(req.ShardID)
	if err != nil {
		return nil, err
	}

	node, err := mgr.Chain.Tip()
	if err != nil {
		return nil, err
	}

	// TODO: add calculated slot here in addition to tip slot
	return &pb.SlotNumberResponse{
		BlockHash: node.BlockHash[:],
		TipSlot: node.Slot,
	}, nil
}

// GetActionStream gets an action stream.
func (s *ShardRPCServer) GetActionStream(req *pb.ShardActionStreamRequest, res pb.ShardRPC_GetActionStreamServer) error {
	manager, err := s.sm.GetManager(req.ShardID)
	if err != nil {
		return err
	}

	n := NewActionStreamGenerator(func(action *pb.ShardChainAction) {
		_ = res.Send(action)
	}, s.config)
	manager.RegisterNotifee(n)
	<- res.Context().Done()
	manager.UnregisterNotifee(n)

	return nil
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


// AnnounceProposal instructs the shard module to subscribe to a specific shard and start downloading blocks. This is a
// no-op until we implement the P2P network.
func (s *ShardRPCServer) AnnounceProposal(ctx context.Context, req *pb.ProposalAnnouncement) (*empty.Empty, error) {
	blockHash, err := chainhash.NewHash(req.BlockHash)
	if err != nil {
		return nil, err
	}

	var mgr *chain.ShardManager

	if !s.sm.IsManaging(req.ShardID) {
		manager, err := s.sm.StartManaging(req.ShardID, chain.ShardChainInitializationParameters{
			RootBlockHash: *blockHash,
			RootSlot:      req.CrosslinkSlot,
		})
		if err != nil {
			return nil, err
		}

		mgr = manager
	} else {
		manager, err := s.sm.GetManager(req.ShardID)
		if err != nil {
			return nil, err
		}

		mgr = manager
	}

	stateHash, err := chainhash.NewHash(req.StateHash)
	if err != nil {
		return nil, err
	}

	mgr.SyncManager.ProposalAnnounced(req.ProposalSlot, *stateHash, req.CrosslinkSlot)

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
			StateHash: primitives.EmptyTree[:],
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
		StateHash: blockNode.StateRoot[:],
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

	proposalInfo := manager.SyncManager.GetProposalInformation()

	txPackage := proposalInfo.TransactionPackage

	transactionRoot, _ := ssz.HashTreeRoot(txPackage.Transactions)

	// for now, block is empty, but we'll fill this in eventually
	block := &primitives.ShardBlock{
		Header: primitives.ShardBlockHeader{
			PreviousBlockHash:   tipNode.BlockHash,
			Slot:                req.Slot,
			Signature:           [48]byte{},
			StateRoot:           txPackage.EndRoot,
			TransactionRoot:     transactionRoot,
			FinalizedBeaconHash: *finalizedBeaconHash,
		},
		Body: primitives.ShardBlockBody{
			Transactions: txPackage.Transactions,
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

	err = manager.ProcessBlock(*block)
	if err != nil {
		return nil, err
	}

	blockBytes, err := proto.Marshal(req.Block)
	if err != nil {
		return nil, err
	}

	err = s.hn.Broadcast(fmt.Sprintf("shard %d blocks", req.Shard), blockBytes)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

var _ pb.ShardRPCServer = &ShardRPCServer{}

// Serve serves the RPC server
func Serve(proto string, listenAddr string, mux *chain.ShardMux, c *config.Config, node *p2p.HostNode) error {
	lis, err := net.Listen(proto, listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterShardRPCServer(s, &ShardRPCServer{mux, node, c})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
