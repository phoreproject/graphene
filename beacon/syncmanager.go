package beacon

import (
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	logger "github.com/sirupsen/logrus"
	"time"
)

// SyncManager is responsible for requesting blocks from clients
// and determining whether the client should be considered in
// sync (for attached validators).
//
// The basic algorithm for syncing is as follows:
// func onReceive(newBlock):
//     if haveBlock(newBlock):
//         processBlock(newBlock)
//     else:
//         locator = generateLocator(chain) - locator starts from tip and goes back exponentially including the genesis block
//				 askForBlocks(locator)
type SyncManager struct {
	hostNode    *p2p.HostNode
	timeout     time.Duration
	syncStarted bool
	blockchain  *Blockchain
}

// NewSyncManager creates a new sync manager
func NewSyncManager(hostNode *p2p.HostNode, timeout time.Duration, blockchain *Blockchain) SyncManager {
	return SyncManager{
		hostNode:    hostNode,
		timeout:     timeout,
		syncStarted: false,
		blockchain:  blockchain,
	}
}

// Connected returns whether the client should be considered connected to the
// network.
func (s SyncManager) Connected() bool {
	peers := s.hostNode.GetPeerList()

	timeSinceLastPing := s.timeout

	for _, p := range peers {
		timeSince := time.Since(p.LastPingTime)

		if timeSince < timeSinceLastPing {
			timeSinceLastPing = timeSince
		}
	}

	return timeSinceLastPing < s.timeout
}

const limitBlocksToSend = 500

func (s SyncManager) onMessageGetBlock(peer *p2p.Peer, message proto.Message) error {
	getBlockMesssage := message.(*pb.GetBlockMessage)

	stopHash, err := chainhash.NewHash(getBlockMesssage.HashStop)
	if err != nil {
		return err
	}

	firstCommonBlock := s.blockchain.View.Chain.Genesis()

	// find the first block that the peer has in our main chain
	for _, h := range getBlockMesssage.LocatorHashes {
		blockHash, err := chainhash.NewHash(h)
		if err != nil {
			return err
		}

		blockNode := s.blockchain.View.Index.GetBlockNodeByHash(*blockHash)
		if blockNode != nil {
			if s.blockchain.View.Chain.Contains(blockNode) {
				firstCommonBlock = blockNode
				break
			}

			tip := s.blockchain.View.Chain.Tip()
			if blockNode.GetAncestorAtHeight(s.blockchain.Height()) == tip {
				firstCommonBlock = tip
			}
		}
	}

	currentBlockNode := s.blockchain.View.Chain.Next(firstCommonBlock)
	if currentBlockNode != nil {
		logger.WithField("common", currentBlockNode.Hash).Debug("found first common block")
	} else {
		logger.Debug("first common block is tip")
		return nil
	}
	toSend := make([]primitives.Block, 0, limitBlocksToSend)

	for currentBlockNode != nil && len(toSend) < limitBlocksToSend && !currentBlockNode.Hash.IsEqual(stopHash) {
		currentBlock, err := s.blockchain.db.GetBlockForHash(currentBlockNode.Hash)
		if err != nil {
			return err
		}

		toSend = append(toSend, *currentBlock)

		currentBlockNode = s.blockchain.View.Chain.Next(currentBlockNode)
	}

	blockMessage := &pb.BlockMessage{
		Blocks: make([]*pb.Block, len(toSend)),
	}

	for i := range toSend {
		blockMessage.Blocks[i] = toSend[i].ToProto()
	}

	return peer.SendMessage(blockMessage)
}

func (s SyncManager) onMessageBlock(peer *p2p.Peer, message proto.Message) error {
	// TODO: limits for this and only should receive blocks if requested

	blockMessage := message.(*pb.BlockMessage)

	logger.WithField("number", len(blockMessage.Blocks)).Debug("received block")

	for i := range blockMessage.Blocks {
		block, err := primitives.BlockFromProto(blockMessage.Blocks[i])
		if err != nil {
			return err
		}

		err = s.handleReceivedBlock(block, peer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s SyncManager) handleReceivedBlock(block *primitives.Block, peerFrom *p2p.Peer) error {
	blockHash, err := ssz.TreeHash(block)
	if err != nil {
		return err
	}

	if s.blockchain.View.Index.GetBlockNodeByHash(block.BlockHeader.ParentRoot) == nil {
		// if we haven't handshaked with them, give up
		if peerFrom == nil {
			return nil
		}

		if _, found := peerFrom.BlocksRequested[blockHash]; found {
			// we already requested this block, so request the parent
			err := peerFrom.SendMessage(&pb.GetBlockMessage{
				LocatorHashes: s.blockchain.View.Chain.GetChainLocator(),
				HashStop:      block.BlockHeader.ParentRoot[:],
			})
			if err != nil {
				return err
			}
		} else {
			// request all blocks up to this block
			err := peerFrom.SendMessage(&pb.GetBlockMessage{
				LocatorHashes: s.blockchain.View.Chain.GetChainLocator(),
				HashStop:      blockHash[:],
			})
			if err != nil {
				return err
			}
		}

	} else {
		err := s.blockchain.ProcessBlock(block, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// ListenForBlocks listens for new blocks over the pub-sub network
// being broadcast as a result of finding them.
func (s SyncManager) ListenForBlocks() error {
	_, err := s.hostNode.SubscribeMessage("block", func(data []byte, from peer.ID) {
		peerFrom := s.hostNode.GetPeerByID(from)

		blockProto := new(pb.Block)

		err := proto.Unmarshal(data, blockProto)
		if err != nil {
			logger.Error(err)
			return
		}

		block, err := primitives.BlockFromProto(blockProto)
		if err != nil {
			logger.Error(err)
			return
		}

		// TODO: ignore new blocks if syncing

		err = s.handleReceivedBlock(block, peerFrom)
		if err != nil {
			logger.Error(err)
			return
		}
	})
	if err != nil {
		return err
	}

	return nil
}

// Start starts the sync manager by registering message handlers
func (s SyncManager) Start() {
	s.hostNode.RegisterMessageHandler("pb.GetBlockMessage", s.onMessageGetBlock)

	s.hostNode.RegisterMessageHandler("pb.BlockMessage", s.onMessageBlock)
}

// TryInitialSync tries to select a peer to sync with and
// start downloading blocks.
func (s SyncManager) TryInitialSync() {
	logger.Debug("INITIAL")

	peers := s.hostNode.GetPeerList()
	if len(peers) == 0 {
		return
	}

	if !s.syncStarted {
		s.syncStarted = true

		bestPeer := peers[0]

		for _, p := range peers[1:] {
			if p.Outbound && !bestPeer.Outbound {
				bestPeer = p
			}
		}

		getBlockMessage := &pb.GetBlockMessage{
			LocatorHashes: s.blockchain.View.Chain.GetChainLocator(),
			HashStop:      zeroHash[:],
		}

		err := bestPeer.SendMessage(getBlockMessage)

		if err != nil {
			logger.Error(err)
			return
		}
	}
}
