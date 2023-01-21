package chain

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	relayerp2p "github.com/phoreproject/synapse/relayer/p2p"
	"github.com/phoreproject/synapse/shard/state"
	"github.com/phoreproject/synapse/shard/transfer"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

var log = logrus.New().WithField("module", "shard.chain")

func init() {
	log.Logger.SetLevel(logrus.DebugLevel)
}

type shardProtocols struct {
	shard   *p2p.ProtocolHandler
	relayer *p2p.ProtocolHandler
}

// SlotProposalManager keeps track of proposals for a specific slot.
type SlotProposalManager struct {
	proposalSlot uint64

	bestProposal *ProposalInformation
	lock         sync.Mutex
	mgr          *ShardSyncManager
	requested    bool
}

// NewSlotProposalManager creates a new manager to keep track of proposals for this slot.
func NewSlotProposalManager(slot uint64, epochState chainhash.Hash, epochSlot uint64, mgr *ShardSyncManager) *SlotProposalManager {
	return &SlotProposalManager{
		proposalSlot: slot,
		bestProposal: &ProposalInformation{
			TransactionPackage: primitives.TransactionPackage{
				StartRoot:     epochState,
				EndRoot:       epochState,
				Updates:       nil,
				Verifications: nil,
				Transactions:  nil,
			},
			startSlot: epochSlot,
		},
		mgr:       mgr,
		requested: false,
	}
}

// RequestPackages requests packages if needed
func (s *SlotProposalManager) RequestPackages(stateRoot chainhash.Hash) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.requestPackages(stateRoot)
}

func (s *SlotProposalManager) requestPackages(stateRoot chainhash.Hash) {
	if s.requested {
		return
	}
	log.WithField("slot", s.proposalSlot).WithField("stateRoot", stateRoot.String()).Debug("requesting for slot")

	s.requested = true

	msg := pb.GetPackagesMessage{
		TipStateRoot: stateRoot[:],
	}

	msgBytes, err := proto.Marshal(&msg)
	if err != nil {
		logrus.Warn(err)
		return
	}

	err = s.mgr.hostNode.Broadcast(fmt.Sprintf("/packageRequests/%d", s.mgr.shardID), msgBytes)
	if err != nil {
		logrus.Warn(err)
	}
}

func (s *SlotProposalManager) requestPackagesAtSlotBoundary() {
	waitUntil := uint64(s.mgr.manager.Config.SlotDuration)*s.proposalSlot + s.mgr.manager.InitializationParameters.GenesisTime
	time.Sleep(time.Until(time.Unix(int64(waitUntil), 0)))

	s.lock.Lock()
	defer s.lock.Unlock()

	startState, err := s.mgr.manager.Chain.GetNodeBySlot(s.proposalSlot - 1)
	if err != nil {
		logrus.Warn(err)
	}

	s.requestPackages(startState.StateRoot)
}

func (s *SlotProposalManager) getProposal() *ProposalInformation {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.bestProposal
}

func (s *SlotProposalManager) processIncomingProposal(p *ProposalInformation) {
	if p.startSlot >= s.proposalSlot {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if p.isBetterThan(s.bestProposal) {
		s.bestProposal = p
	}
}

// ShardSyncManager is a manager to help sync up the shard chain.
type ShardSyncManager struct {
	hostNode *p2p.HostNode

	manager   *ShardManager
	protocols shardProtocols
	shardID   uint64

	slotProposals map[uint64]*SlotProposalManager
}

// AddBlock is a notifee from the manager when a block is added.
func (s *ShardSyncManager) AddBlock(block *primitives.ShardBlock, newTip bool) {
	slot := block.Header.Slot

	if !newTip {
		return
	}

	for proposalSlot, mgr := range s.slotProposals {
		mgr.lock.Lock()
		if slot < proposalSlot && !mgr.bestProposal.TransactionPackage.StartRoot.IsEqual(&block.Header.StateRoot) {
			mgr.bestProposal = &ProposalInformation{
				TransactionPackage: primitives.TransactionPackage{
					StartRoot:     block.Header.StateRoot,
					EndRoot:       block.Header.StateRoot,
					Updates:       nil,
					Verifications: nil,
					Transactions:  nil,
				},
				startSlot: block.Header.Slot,
			}
		}
		mgr.lock.Unlock()
	}

	for proposalSlot, mgr := range s.slotProposals {
		if slot >= proposalSlot-1 {
			// TODO: improve?
			time.Sleep(200 * time.Millisecond)

			mgr.RequestPackages(block.Header.StateRoot)
		}
	}
}

// FinalizeBlockHash is an unused notifee from the manager when a block is finalized.
func (s *ShardSyncManager) FinalizeBlockHash(blockHash chainhash.Hash, slot uint64) {}

// PeerConnected is called when a peer connects to the shard module.
func (s *ShardSyncManager) PeerConnected(id peer.ID, dir network.Direction) {
	// if the peer is inbound, we should send them a version message
	if dir == network.DirInbound {
		err := s.protocols.shard.SendMessage(id, &pb.ShardVersionMessage{
			Version: 0,
			Height:  s.manager.Chain.Height(),
		})
		if err != nil {
			logrus.Error(err)
			return
		}
	}
}

// ProposalAnnounced registers a slot to be proposed.
func (s *ShardSyncManager) ProposalAnnounced(shardProposals []uint64, startState chainhash.Hash, startSlot uint64) {
	// worst case, we propose an empty block
	oldSlotProposals := s.slotProposals
	newProposals := make(map[uint64]*SlotProposalManager)

	for _, slot := range shardProposals {
		root := startState
		stateSlot := startSlot
		if manager, found := oldSlotProposals[slot]; found {
			root = manager.bestProposal.TransactionPackage.StartRoot
			stateSlot = manager.bestProposal.startSlot
		}

		newProposals[slot] = NewSlotProposalManager(slot, root, stateSlot, s)

		currentSlot := slot
		proposalManager, found := oldSlotProposals[currentSlot]
		for !found && currentSlot > 0 {
			currentSlot--
			proposalManager, found = oldSlotProposals[currentSlot]
		}

		if found {
			newProposals[slot].bestProposal = proposalManager.bestProposal
		}
	}

	s.slotProposals = newProposals
}

// ProposalInformation is information about how a validator should propose a block.
type ProposalInformation struct {
	TransactionPackage primitives.TransactionPackage
	startSlot          uint64
}

func (p *ProposalInformation) isBetterThan(p2 *ProposalInformation) bool {
	if p2.startSlot > p.startSlot {
		return false
	}

	// TODO: better heuristic for which package of transactions is better
	if len(p2.TransactionPackage.Transactions) > len(p.TransactionPackage.Transactions) {
		return false
	}

	return true
}

// GetProposalInformation gets the proposal information including the package.
func (s *ShardSyncManager) GetProposalInformation(forSlot uint64) *ProposalInformation {
	// TODO: cleanup subscriptions here
	return s.slotProposals[forSlot].getProposal()
}

// PeerDisconnected is called when a peer disconnects from the shard module.
func (s *ShardSyncManager) PeerDisconnected(peer.ID) {}

// NewShardSyncManager creates a new shard sync manager.
func NewShardSyncManager(hn *p2p.HostNode, manager *ShardManager, shardID uint64) (*ShardSyncManager, error) {

	sm := &ShardSyncManager{
		hostNode: hn,
		manager:  manager,
		shardID:  shardID,
	}

	manager.RegisterNotifee(sm)

	if err := sm.registerP2P(); err != nil {
		return nil, err
	}

	return sm, nil
}

// Unregister is called when we no longer care about this shard.
func (s *ShardSyncManager) Unregister() error {
	return nil
}

var zeroHash = chainhash.Hash{}

func (s *ShardSyncManager) onMessageVersion(id peer.ID, msg proto.Message) error {
	getVersionMessage := msg.(*pb.ShardVersionMessage)

	if s.hostNode.GetPeerDirection(id) == network.DirInbound {
		err := s.protocols.shard.SendMessage(id, &pb.ShardVersionMessage{
			Version: 0,
			Height:  s.manager.Chain.Height(),
		})
		if err != nil {
			return err
		}
	}

	if getVersionMessage.Height >= s.manager.Chain.Height() {
		locatorHashes, err := s.manager.Chain.GetChainLocator()
		if err != nil {
			return err
		}
		err = s.protocols.shard.SendMessage(id, &pb.GetShardBlocksMessage{
			LocatorHashes: locatorHashes,
			HashStop:      zeroHash[:],
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ShardSyncManager) onMessagePackage(id peer.ID, msg proto.Message) error {
	packageMessage := msg.(*pb.PackageMessage)

	log.WithField("numTransactions", len(packageMessage.Package.Transactions)).
		WithField("startHash", packageMessage.Package.StartRoot).
		WithField("endHash", packageMessage.Package.EndRoot).
		Debug("received package message")

	transactionPackage, err := primitives.TransactionPackageFromProto(packageMessage.Package)
	if err != nil {
		return err
	}

	slotRoot, err := s.manager.Chain.GetLatestNodeBySlot(packageMessage.StartSlot)
	if err != nil {
		return err
	}

	if !slotRoot.StateRoot.IsEqual(&transactionPackage.StartRoot) {
		return fmt.Errorf("start root doesn't match package (got: %s, expected: %s, slot: %d)", transactionPackage.StartRoot, slotRoot.StateRoot, packageMessage.StartSlot)
	}

	p := ProposalInformation{
		TransactionPackage: *transactionPackage,
		startSlot:          packageMessage.StartSlot,
	}

	witnessStateProvider := state.NewPartialShardState(transactionPackage.StartRoot, transactionPackage.Verifications, transactionPackage.Updates)

	currentRoot := &transactionPackage.StartRoot
	for _, tx := range transactionPackage.Transactions {
		newRoot, err := state.Transition(witnessStateProvider, tx.TransactionData, state.ShardInfo{
			CurrentCode: transfer.Code,
			ShardID:     uint32(s.shardID),
		})
		if err != nil {
			return err
		}
		currentRoot = newRoot
	}

	if !currentRoot.IsEqual(&transactionPackage.EndRoot) {
		return fmt.Errorf("end root (%s) does not match expected end root (%s)", currentRoot, transactionPackage.EndRoot)
	}
	for _, slotManager := range s.slotProposals {
		slotManager.processIncomingProposal(&p)
	}

	return nil
}

func (s *ShardSyncManager) onMessageBlock(id peer.ID, msg proto.Message) error {
	blockMessage := msg.(*pb.ShardBlockMessage)

	logrus.WithFields(logrus.Fields{
		"from":   blockMessage.Blocks[0].Header.Slot,
		"to":     blockMessage.Blocks[len(blockMessage.Blocks)-1].Header.Slot,
		"number": len(blockMessage.Blocks),
	}).Debug("received blocks from sync")

	logrus.Debug("checking signatures")

	if len(blockMessage.Blocks) == 0 {
		// TODO: handle error of peer sending no blocks
		return nil
	}

	blocks := make([]*primitives.ShardBlock, len(blockMessage.Blocks))
	for i := range blockMessage.Blocks {
		b, err := primitives.ShardBlockFromProto(blockMessage.Blocks[i])
		if err != nil {
			return err
		}
		blocks[i] = b
	}

	for _, b := range blocks {
		err := s.handleReceivedBlock(b, id)
		if err != nil {
			return err
		}
	}

	lastBlockHash, err := ssz.HashTreeRoot(blockMessage.Blocks[len(blockMessage.Blocks)-1])
	if err != nil {
		return err
	}

	if !bytes.Equal(lastBlockHash[:], blockMessage.LatestBlockHash) && !bytes.Equal(blockMessage.LatestBlockHash, zeroHash[:]) {
		logrus.Infof("continuing sync to block %x", blockMessage.LatestBlockHash)

		loc, err := s.manager.Chain.GetChainLocator()
		if err != nil {
			return err
		}

		// request all blocks up to this block
		err = s.protocols.shard.SendMessage(id, &pb.GetShardBlocksMessage{
			LocatorHashes: loc,
			HashStop:      blockMessage.LatestBlockHash,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ShardSyncManager) onMessageGetShardBlocks(id peer.ID, msg proto.Message) error {
	getBlockMessage := msg.(*pb.GetShardBlocksMessage)

	stopHash, err := chainhash.NewHash(getBlockMessage.HashStop)
	if err != nil {
		return err
	}

	firstCommonBlock := s.manager.Chain.Genesis()

	if len(getBlockMessage.LocatorHashes) == 0 {
		return nil
	}

	if !bytes.Equal(firstCommonBlock.BlockHash[:], getBlockMessage.LocatorHashes[len(getBlockMessage.LocatorHashes)-1]) {
		return nil
	}

	for _, h := range getBlockMessage.LocatorHashes {
		blockHash, err := chainhash.NewHash(h)
		if err != nil {
			return err
		}

		blockNode, err := s.manager.Index.GetNodeByHash(blockHash)
		if err != nil {
			return err
		}

		if s.manager.Chain.Contains(blockNode) {
			firstCommonBlock = blockNode
			break
		}

		tip, err := s.manager.Chain.Tip()
		if err != nil {
			return err
		}
		if blockNode.GetAncestorAtSlot(s.manager.Chain.Height()) == tip {
			firstCommonBlock = tip
		}
	}

	currentBlockNode := s.manager.Chain.Next(firstCommonBlock)

	if currentBlockNode != nil {
		logrus.WithField("common", currentBlockNode.BlockHash).Debug("found first common block")
	} else {
		logrus.Debug("first common block is tip")
		return nil
	}

	const limitBlocksToSend = 500
	toSend := make([]*pb.ShardBlock, 0, limitBlocksToSend)

	for currentBlockNode != nil && len(toSend) < limitBlocksToSend {
		blockToSend, err := s.manager.BlockDB.GetBlockForHash(currentBlockNode.BlockHash)
		if err != nil {
			return err
		}

		toSend = append(toSend, blockToSend.ToProto())

		if currentBlockNode.BlockHash.IsEqual(stopHash) {
			break
		}

		currentBlockNode = s.manager.Chain.Next(currentBlockNode)
	}

	if len(toSend) > 0 {
		logrus.WithFields(logrus.Fields{
			"from": firstCommonBlock.Slot,
			"to":   int(firstCommonBlock.Slot) + len(toSend),
		}).Debug("sending blocks to peer")
	}

	tipHash, err := s.manager.Chain.Tip()
	if err != nil {
		return err
	}

	blockMessage := &pb.ShardBlockMessage{
		Blocks:          toSend,
		LatestBlockHash: tipHash.BlockHash[:],
	}

	return s.protocols.shard.SendMessage(id, blockMessage)
}

func (s *ShardSyncManager) processBlock(blockBytes []byte, id peer.ID) {
	var blockProto pb.ShardBlock
	err := proto.Unmarshal(blockBytes, &blockProto)
	if err != nil {
		logrus.Error(err)
		return
	}

	block, err := primitives.ShardBlockFromProto(&blockProto)
	if err != nil {
		logrus.Error(err)
	}
	blockHash, _ := ssz.HashTreeRoot(block)

	if s.manager.Index.HasBlock(blockHash) {
		// we already have this block
		return
	}

	if err := s.handleReceivedBlock(block, id); err != nil {
		logrus.Error(err)
	}
}

func (s *ShardSyncManager) handleReceivedBlock(block *primitives.ShardBlock, peerFrom peer.ID) error {
	blockHash, err := ssz.HashTreeRoot(block)
	if err != nil {
		return err
	}

	_, err = s.manager.Index.GetNodeByHash(&block.Header.PreviousBlockHash)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"hash":       chainhash.Hash(block.Header.PreviousBlockHash),
			"slotTrying": block.Header.Slot,
		}).Info("requesting parent block")

		loc, err := s.manager.Chain.GetChainLocator()
		if err != nil {
			return err
		}

		// request all blocks up to this block
		err = s.protocols.shard.SendMessage(peerFrom, &pb.GetBlocksMessage{
			LocatorHashes: loc,
			HashStop:      blockHash[:],
		})
		if err != nil {
			return err
		}

	} else {
		err := s.manager.ProcessBlock(*block)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ShardSyncManager) registerP2P() error {
	// TODO: all this needs to be cleaned up eventually

	relayer, err := s.hostNode.RegisterProtocolHandler(relayerp2p.RelayerSyncProtocol(s.shardID, 1), 5, 1)
	if err != nil {
		return err
	}

	shardBlocks, err := s.hostNode.RegisterProtocolHandler(ShardSyncProtocol(s.shardID, 1), 5, 1)
	if err != nil {
		return err
	}

	shardBlocks.Notify(s)

	_, err = s.hostNode.SubscribeMessage(fmt.Sprintf("shard %d blocks", s.shardID), s.processBlock)
	if err != nil {
		return err
	}

	err = shardBlocks.RegisterHandler("pb.GetShardBlocksMessage", s.onMessageGetShardBlocks)
	if err != nil {
		return err
	}

	err = shardBlocks.RegisterHandler("pb.ShardVersionMessage", s.onMessageVersion)
	if err != nil {
		return err
	}

	err = shardBlocks.RegisterHandler("pb.ShardBlockMessage", s.onMessageBlock)
	if err != nil {
		return err
	}

	err = relayer.RegisterHandler("pb.PackageMessage", s.onMessagePackage)
	if err != nil {
		return err
	}

	s.protocols = shardProtocols{
		relayer: relayer,
		shard:   shardBlocks,
	}

	return nil
}

// ShardSyncProtocol gets the protocol ID for a specific shard and version.
func ShardSyncProtocol(shardID uint64, version uint64) protocol.ID {
	return protocol.ID(fmt.Sprintf("/phore/shard/%d/v%d", shardID, version))
}
