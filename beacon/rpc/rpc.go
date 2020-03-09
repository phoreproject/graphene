package rpc

import (
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/go-ssz"

	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/utils"

	"github.com/golang/protobuf/proto"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/chainhash"

	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// server is used to implement rpc.BlockchainRPCServer.
type server struct {
	chain   *beacon.Blockchain
	p2p     *p2p.HostNode
	mempool *beacon.Mempool
}

// GetGenesisTime gets the genesis time.
func (s *server) GetGenesisTime(context.Context, *empty.Empty) (*pb.GenesisTimeResponse, error) {
	return &pb.GenesisTimeResponse{
		GenesisTime: s.chain.GetGenesisTime(),
	}, nil
}

// GetListeningAddresses gets the addresses we're listening on.
func (s *server) GetListeningAddresses(context.Context, *empty.Empty) (*pb.ListeningAddressesResponse, error) {
	addrs := s.p2p.GetHost().Addrs()

	info := peer.AddrInfo{
		ID:    s.p2p.GetHost().ID(),
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
		Addresses: addrStrings,
	}, nil
}

// Connect attempts to connect to a node.
func (s *server) Connect(ctx context.Context, connectMsg *pb.ConnectMessage) (*empty.Empty, error) {
	addr := connectMsg.Address

	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}

	pi, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return nil, err
	}

	err = s.p2p.Connect(ctx, *pi)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

// GetShardProposerForSlot gets the shard proposer ID and public key for a certain slot on a certain shard.
func (s *server) GetShardProposerForSlot(ctx context.Context, req *pb.GetShardProposerRequest) (*pb.ShardProposerResponse, error) {
	state := s.chain.GetState()
	proposer, err := state.GetShardProposerIndex(req.Slot, req.ShardID, s.chain.GetConfig())
	if err != nil {
		return nil, err
	}

	proposerPubKey := state.ValidatorRegistry[proposer].Pubkey

	return &pb.ShardProposerResponse{
		Proposer:          proposer,
		ProposerPublicKey: proposerPubKey[:],
	}, nil
}

// SubmitAttestation submits an attestation to the mempool.
func (s *server) SubmitAttestation(ctx context.Context, att *pb.Attestation) (*empty.Empty, error) {
	a, err := primitives.AttestationFromProto(att)
	if err != nil {
		return nil, err
	}
	err = s.mempool.ProcessNewAttestation(*a)
	if err != nil {
		return nil, err
	}

	data, err := proto.Marshal(att)
	if err != nil {
		return nil, err
	}

	err = s.p2p.Broadcast("attestation", data)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

// GetMempool gets the mempool for a block.
func (s *server) GetMempool(ctx context.Context, req *pb.MempoolRequest) (*pb.BlockBody, error) {
	lastBlockHash, err := chainhash.NewHash(req.LastBlockHash)
	if err != nil {
		return nil, err
	}

	atts, err := s.mempool.GetAttestationsToInclude(s.chain.GetCurrentSlot(), *lastBlockHash, s.chain.GetConfig())
	if err != nil {
		return nil, err
	}

	bb := primitives.BlockBody{
		Attestations:      atts,
		ProposerSlashings: make([]primitives.ProposerSlashing, 0),
		CasperSlashings:   make([]primitives.CasperSlashing, 0),
		Deposits:          make([]primitives.Deposit, 0),
		Exits:             make([]primitives.Exit, 0),
	}

	return bb.ToProto(), nil
}

// SubmitBlock submits a block to the network after verifying it
func (s *server) SubmitBlock(ctx context.Context, in *pb.SubmitBlockRequest) (*pb.SubmitBlockResponse, error) {
	b, err := primitives.BlockFromProto(in.Block)
	if err != nil {
		return nil, err
	}

	_, _, err = s.chain.ProcessBlock(b, true, true)
	if err != nil {
		return nil, err
	}
	h, err := ssz.HashTreeRoot(b)
	if err != nil {
		return nil, err
	}

	data, err := proto.Marshal(b.ToProto())
	if err != nil {
		return nil, err
	}

	err = s.p2p.Broadcast("block", data)
	if err != nil {
		return nil, err
	}
	return &pb.SubmitBlockResponse{BlockHash: h[:]}, err
}

// GetSlotNumber gets the current slot number.
func (s *server) GetSlotNumber(ctx context.Context, in *empty.Empty) (*pb.SlotNumberResponse, error) {
	state := s.chain.GetState()
	config := s.chain.GetConfig()
	genesisTime := state.GenesisTime
	timePerSlot := config.SlotDuration
	currentTime := utils.Now().Unix()
	currentSlot := (currentTime - int64(genesisTime)) / int64(timePerSlot)
	if currentSlot < 0 {
		currentSlot = 0
	}
	block := s.chain.View.Chain.GetBlockBySlot(uint64(currentSlot))
	return &pb.SlotNumberResponse{SlotNumber: uint64(currentSlot), BlockHash: block.Hash[:], TipSlot: block.Slot}, nil
}

// GetBlockHash gets the block hash for a certain slot in the main chain.
func (s *server) GetBlockHash(ctx context.Context, in *pb.GetBlockHashRequest) (*pb.GetBlockHashResponse, error) {
	n := s.chain.View.Chain.GetBlockBySlot(in.SlotNumber)
	return &pb.GetBlockHashResponse{Hash: n.Hash[:]}, nil
}

// GetLastBlockHash gets the most recent block hash in the main chain.
func (s *server) GetLastBlockHash(ctx context.Context, in *empty.Empty) (*pb.GetBlockHashResponse, error) {
	h := s.chain.View.Chain.Tip()
	return &pb.GetBlockHashResponse{
		Hash: h.Hash[:],
	}, nil
}

// GetState gets the state of the main chain.
func (s *server) GetState(ctx context.Context, in *empty.Empty) (*pb.GetStateResponse, error) {
	state := s.chain.GetState()
	stateProto := state.ToProto()

	return &pb.GetStateResponse{State: stateProto}, nil
}

// GetStateRoot gets the hash of the state in the main chain.
func (s *server) GetStateRoot(ctx context.Context, in *empty.Empty) (*pb.GetStateRootResponse, error) {
	return &pb.GetStateRootResponse{StateRoot: s.chain.View.Chain.Tip().StateRoot[:]}, nil
}

// GetEpochInformation gets information about the current epoch used for attestation
// assignment and generation.
func (s *server) GetEpochInformation(ctx context.Context, in *pb.EpochInformationRequest) (*pb.EpochInformationResponse, error) {
	state := s.chain.GetState()
	config := s.chain.GetConfig()

	requestedEpochSlot := uint64(in.EpochIndex) * s.chain.GetConfig().EpochLength

	if requestedEpochSlot > state.Slot {
		updatedState, err := s.chain.GetUpdatedState(requestedEpochSlot)
		if err != nil {
			return nil, err
		}
		state = *updatedState
	}

	if state.EpochIndex < in.EpochIndex {
		state = state.Copy()

		_, err := state.ProcessEpochTransition(s.chain.GetConfig())
		if err != nil {
			return nil, err
		}
	}

	epochBoundaryRoot := s.chain.View.Chain.GetBlockBySlot(in.EpochIndex * config.EpochLength)
	previousEpochBoundaryRoot := s.chain.View.Chain.GetBlockBySlot((in.EpochIndex - 1) * config.EpochLength)

	latestCrosslinks := make([]*pb.Crosslink, len(state.LatestCrosslinks))
	for i := range latestCrosslinks {
		latestCrosslinks[i] = state.LatestCrosslinks[i].ToProto()
	}

	previousCrosslinks := make([]*pb.Crosslink, len(state.PreviousCrosslinks))
	for i := range previousCrosslinks {
		previousCrosslinks[i] = state.PreviousCrosslinks[i].ToProto()
	}

	justifiedNode := s.chain.View.Chain.GetBlockBySlot(state.JustifiedEpoch * config.EpochLength)
	previousJustifiedNode := s.chain.View.Chain.GetBlockBySlot(state.PreviousJustifiedEpoch * config.EpochLength)

	earliestSlot := int64(state.Slot) - int64(state.Slot%config.EpochLength) - int64(config.EpochLength)

	newShuffling := state.GetAssignmentsAssumingShuffle(config)

	slotsAssumingShuffle := make([]*pb.ShardCommitteesForSlot, len(newShuffling))
	for s := range slotsAssumingShuffle {
		shardAndCommittees := make([]*pb.ShardCommittee, len(newShuffling[s]))
		for i := range shardAndCommittees {
			shardAndCommittees[i] = newShuffling[s][i].ToProto()
		}
		slotsAssumingShuffle[s] = &pb.ShardCommitteesForSlot{
			Committees: shardAndCommittees,
		}
	}

	slots := make([]*pb.ShardCommitteesForSlot, len(state.ShardAndCommitteeForSlots))
	for s := range slots {
		shardAndCommittees := make([]*pb.ShardCommittee, len(state.ShardAndCommitteeForSlots[s]))
		for i := range shardAndCommittees {
			shardAndCommittees[i] = state.ShardAndCommitteeForSlots[s][i].ToProto()
		}
		slots[s] = &pb.ShardCommitteesForSlot{
			Committees: shardAndCommittees,
		}
	}

	return &pb.EpochInformationResponse{
		HasEpochInformation: true,
		Information: &pb.EpochInformation{
			ShardCommitteesForSlots:     slots,
			ShardCommitteesForNextEpoch: slotsAssumingShuffle,
			Slot:                        earliestSlot,
			TargetHash:                  epochBoundaryRoot.Hash[:],
			JustifiedEpoch:              state.JustifiedEpoch,
			LatestCrosslinks:            latestCrosslinks,
			PreviousCrosslinks:          previousCrosslinks,
			JustifiedHash:               justifiedNode.Hash[:],
			PreviousTargetHash:          previousEpochBoundaryRoot.Hash[:],
			PreviousJustifiedEpoch:      state.PreviousJustifiedEpoch,
			PreviousJustifiedHash:       previousJustifiedNode.Hash[:],
		},
	}, nil
}

// GetCommitteesForSlot gets the current committees at a slot.
func (s *server) GetCommitteesForSlot(ctx context.Context, in *pb.GetCommitteesForSlotRequest) (*pb.ShardCommitteesForSlot, error) {
	state := s.chain.GetState()

	sc, err := state.GetShardCommitteesAtSlot(in.Slot, s.chain.GetConfig())
	if err != nil {
		return nil, err
	}
	scProto := make([]*pb.ShardCommittee, len(sc))
	for i := range sc {
		scProto[i] = sc[i].ToProto()
	}

	return &pb.ShardCommitteesForSlot{
		Committees: scProto,
	}, nil
}

// GetForkData gets the current fork data.
func (s *server) GetForkData(ctx context.Context, in *empty.Empty) (*pb.ForkData, error) {
	state := s.chain.GetState()
	return state.ForkData.ToProto(), nil
}

// GetProposerForSlot gets the proposer for a certain slot.
func (s *server) GetProposerForSlot(ctx context.Context, in *pb.GetProposerForSlotRequest) (*pb.GetProposerForSlotResponse, error) {
	state := s.chain.GetState()
	idx, err := state.GetBeaconProposerIndex(in.Slot, s.chain.GetConfig())
	if err != nil {
		return nil, err
	}
	return &pb.GetProposerForSlotResponse{
		Proposer: idx,
	}, nil
}

// GetBlock gets a block by hash.
func (s *server) GetBlock(ctx context.Context, in *pb.GetBlockRequest) (*pb.GetBlockResponse, error) {
	h, err := chainhash.NewHash(in.Hash)
	if err != nil {
		return nil, err
	}
	block, err := s.chain.GetBlockByHash(*h)
	if err != nil {
		return nil, err
	}

	return &pb.GetBlockResponse{
		Block: block.ToProto(),
	}, nil
}

// GetValidatorInformation gets information about a validator.
func (s *server) GetValidatorInformation(ctx context.Context, in *pb.GetValidatorRequest) (*pb.Validator, error) {
	state := s.chain.GetState()

	if uint32(len(state.ValidatorRegistry)) <= in.ID {
		return nil, fmt.Errorf("could not find validator with ID %d", in.ID)
	}

	validator := state.ValidatorRegistry[in.ID]

	return validator.ToProto(), nil
}

// CrosslinkStream creates a crosslink stream.
func (s *server) CrosslinkStream(req *pb.CrosslinkStreamRequest, res pb.BlockchainRPC_CrosslinkStreamServer) error {
	cs := NewCrosslinkStream(req.ShardID, func(c *primitives.Crosslink) {
		_ = res.Send(&pb.CrosslinkMessage{
			BlockHash: c.ShardBlockHash[:],
			Slot:      c.Slot,
		})
	})
	s.chain.RegisterNotifee(cs)
	<-res.Context().Done()
	s.chain.UnregisterNotifee(cs)
	return nil
}

var _ pb.BlockchainRPCServer = &server{}

// Serve serves the RPC server
func Serve(proto string, listenAddr string, b *beacon.Blockchain, hostNode *p2p.HostNode, mempool *beacon.Mempool) error {
	lis, err := net.Listen(proto, listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterBlockchainRPCServer(s, &server{b, hostNode, mempool})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
