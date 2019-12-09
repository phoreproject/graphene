package chain

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	relayerp2p "github.com/phoreproject/synapse/relayer/p2p"
	"github.com/phoreproject/synapse/shard/execution"
	"github.com/phoreproject/synapse/shard/state"
	"github.com/phoreproject/synapse/shard/transfer"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/sirupsen/logrus"
	"sync"
)

type shardProtocols struct {
	shard   *p2p.ProtocolHandler
	relayer *p2p.ProtocolHandler
}

// ShardSyncManager is a manager to help sync up the shard chain.
type ShardSyncManager struct {
	hostNode *p2p.HostNode

	manager   *ShardManager
	protocols shardProtocols
	shardID   uint64

	bestProposal     *ProposalInformation
	bestProposalLock sync.RWMutex
}

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
func (s *ShardSyncManager) ProposalAnnounced(proposalSlot uint64, startState chainhash.Hash, startSlot uint64) {
	// worst case, we propose an empty block
	s.bestProposalLock.Lock()
	defer s.bestProposalLock.Unlock()
	s.bestProposal = &ProposalInformation{
		TransactionPackage: primitives.TransactionPackage{
			StartRoot:     startState,
			EndRoot:       startState,
			Updates:       nil,
			Verifications: nil,
			Transactions:  nil,
		},
		startSlot: startSlot,
	}

	msg := pb.GetPackagesMessage{
		TipStateRoot: startState[:],
	}

	msgBytes, err := proto.Marshal(&msg)
	if err != nil {
		logrus.Warn(err)
		return
	}

	err = s.hostNode.Broadcast(fmt.Sprintf("/packageRequests/%d", s.shardID), msgBytes)
	if err != nil {
		logrus.Warn(err)
	}
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
func (s *ShardSyncManager) GetProposalInformation() *ProposalInformation {
	// TODO: cleanup subscriptions here
	s.bestProposalLock.Lock()
	defer s.bestProposalLock.Unlock()
	return s.bestProposal
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

	transactionPackage, err := primitives.TransactionPackageFromProto(packageMessage.Package)
	if err != nil {
		return err
	}

	slotRoot, err := s.manager.Chain.GetNodeBySlot(packageMessage.StartSlot)
	if err != nil {
		return err
	}

	if !slotRoot.StateRoot.IsEqual(&transactionPackage.StartRoot) {
		return errors.New("start root doesn't match package")
	}

	p := ProposalInformation{
		TransactionPackage: *transactionPackage,
		startSlot:          packageMessage.StartSlot,
	}

	witnessStateProvider := state.NewPartialShardState(transactionPackage.StartRoot, transactionPackage.Verifications, transactionPackage.Updates)

	currentRoot := &transactionPackage.StartRoot
	for _, tx := range transactionPackage.Transactions {
		newRoot, err := execution.Transition(witnessStateProvider, tx.TransactionData, execution.ShardInfo{
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

	s.bestProposalLock.Lock()
	if p.isBetterThan(s.bestProposal) {
		s.bestProposal = &p
	}
	defer s.bestProposalLock.Unlock()

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
		// we already have this blockpbl
		return
	}

	logrus.WithFields(logrus.Fields{
		"slot":  block.Header.Slot,
		"hash":  chainhash.Hash(blockHash),
		"shard": s.shardID,
	}).Debug("got new block from broadcast")

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
