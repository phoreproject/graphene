package beacon

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/prysmaticlabs/go-ssz"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/sirupsen/logrus"
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
	hostNode        *p2p.HostNode
	syncStarted     bool
	blockchain      *Blockchain
	mempool         *Mempool
	postProcessHook func(*primitives.Block, *primitives.State, []primitives.Receipt)

	protocol *p2p.ProtocolHandler
}

// NewSyncManager creates a new sync manager
func NewSyncManager(hostNode *p2p.HostNode, blockchain *Blockchain, mempool *Mempool) (*SyncManager, error) {
	sm := &SyncManager{
		hostNode:    hostNode,
		syncStarted: false,
		blockchain:  blockchain,
		mempool:     mempool,
	}

	ph, err := sm.registerHandlers()
	if err != nil {
		return nil, err
	}

	sm.protocol = ph

	return sm, nil
}

// Connected returns whether the client should be considered connected to the
// network.
func (s SyncManager) Connected() bool {
	return s.hostNode.IsConnected()
}

const limitBlocksToSend = 500

func (s SyncManager) onMessageGetBlock(peer peer.ID, message proto.Message) error {
	//getBlockMesssage := message.(*pb.GetBlockMessage)
	//
	//stopHash, err := chainhash.NewHash(getBlockMesssage.HashStop)
	//if err != nil {
	//	return err
	//}
	//
	//firstCommonBlock := s.blockchain.View.Chain.Genesis()
	//
	//if len(getBlockMesssage.LocatorHashes) == 0 {
	//	// TODO: ban peer
	//	return nil
	//}
	//
	//if !bytes.Equal(firstCommonBlock.Hash[:], getBlockMesssage.LocatorHashes[len(getBlockMesssage.LocatorHashes)-1]) {
	//	// TODO: ban peer
	//	return nil
	//}
	//
	//// find the first block that the peer has in our main chain
	//for _, h := range getBlockMesssage.LocatorHashes {
	//	blockHash, err := chainhash.NewHash(h)
	//	if err != nil {
	//		return err
	//	}
	//
	//	blockNode := s.blockchain.View.Index.GetBlockNodeByHash(*blockHash)
	//	if blockNode != nil {
	//		if s.blockchain.View.Chain.Contains(blockNode) {
	//			firstCommonBlock = blockNode
	//			break
	//		}
	//
	//		tip := s.blockchain.View.Chain.Tip()
	//		if blockNode.GetAncestorAtHeight(s.blockchain.Height()) == tip {
	//			firstCommonBlock = tip
	//		}
	//	}
	//}
	//
	//currentBlockNode := s.blockchain.View.Chain.Next(firstCommonBlock)
	//if currentBlockNode != nil {
	//	logger.WithField("common", currentBlockNode.Hash).Debug("found first common block")
	//} else {
	//	logger.Debug("first common block is tip")
	//	return nil
	//}
	//toSend := make([]string, 0, limitBlocksToSend)
	//
	//for currentBlockNode != nil && len(toSend) < limitBlocksToSend {
	//	s.ipfsAPI.Object().Data()
	//
	//	toSend = append(toSend, currentBlockNode.Hash[:])
	//
	//	if currentBlockNode.Hash.IsEqual(stopHash) {
	//		break
	//	}
	//
	//	currentBlockNode = s.blockchain.View.Chain.Next(currentBlockNode)
	//}
	//
	//if len(toSend) > 0 {
	//	logger.WithFields(logger.Fields{
	//		"from": firstCommonBlock.Slot,
	//		"to":   int(firstCommonBlock.Slot) + len(toSend),
	//	}).Debug("sending blocks to peer")
	//}
	//
	//tipHash := s.blockchain.View.Chain.Tip()
	//
	//blockMessage := &pb.BlockMessage{
	//	BlockPaths:          toSend,
	//	LatestBlockHash: tipHash.Hash[:],
	//}
	//
	//peer.SendMessage(blockMessage)
	//
	return nil
}

// BlockFilter is a filter for block hashes that returns whether the block hash
// is in the filter or not.
type BlockFilter interface {
	Has(chainhash.Hash) bool
}

// 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16, 17 w/ epoch_length 8
// [1 2 3 4 5 6 7 8], [9, 10, 11, 12, 13, 14, 15, 16], [17]

func splitIncomingBlocksIntoChunks(blocks []*primitives.Block, c *config.Config, filter BlockFilter) ([][]*primitives.Block, error) {
	firstBlock := blocks[0]

	epochBlockChunks := make([][]*primitives.Block, 0)
	currentEpochChunk := make([]*primitives.Block, 0)
	currentEpoch := (firstBlock.BlockHeader.SlotNumber + c.EpochLength - 1) / c.EpochLength

	logrus.WithFields(logrus.Fields{
		"firstBlock": blocks[0].BlockHeader.SlotNumber,
		"lastBlock":  blocks[len(blocks)-1].BlockHeader.SlotNumber,
	}).Info("received blocks from peer")

	for _, block := range blocks {
		blockHash, err := ssz.HashTreeRoot(block)
		if err != nil {
			return nil, err
		}

		if filter.Has(blockHash) {
			// ignore blocks we already have.
			continue
		}

		addToNextEpoch := true

		if currentEpoch != (block.BlockHeader.SlotNumber+c.EpochLength-1)/c.EpochLength {
			if block.BlockHeader.SlotNumber%c.EpochLength == 0 {
				currentEpochChunk = append(currentEpochChunk, block)
				addToNextEpoch = false
			}
			if len(currentEpochChunk) > 0 {
				epochBlockChunks = append(epochBlockChunks, currentEpochChunk)
			}
			currentEpochChunk = make([]*primitives.Block, 0)
			currentEpoch = (block.BlockHeader.SlotNumber + c.EpochLength - 1) / c.EpochLength

		}

		// add to chunk if this is not the next epoch
		if addToNextEpoch {
			currentEpochChunk = append(currentEpochChunk, block)
		}
	}

	if len(currentEpochChunk) > 0 {
		epochBlockChunks = append(epochBlockChunks, currentEpochChunk)
	}

	return epochBlockChunks, nil
}

func (s SyncManager) onMessageBlock(peer *peer.ID, message proto.Message) error {
	// TODO: limits for this and only should receive blocks if requested

	//blockMessage := message.(*pb.BlockMessage)

	//ctx := context.Background()
	//
	//for p := blockMessage.BlockPaths {
	//	blockPath, err := path.ParsePath(p)
	//	if err != nil {
	//		return err
	//	}
	//
	//	n, err := s.ipfsAPI.Block().Get(ctx, blockPath)
	//}
	//
	//logger.WithFields(logger.Fields{
	//	"from":   blockMessage.Blocks[0].Header.SlotNumber,
	//	"to":     blockMessage.Blocks[len(blockMessage.Blocks)-1].Header.SlotNumber,
	//	"number": len(blockMessage.Blocks),
	//}).Debug("received blocks from sync")
	//
	//logger.Debug("checking signatures")
	//
	//peer.ProcessingRequest = true
	//
	//if len(blockMessage.Blocks) == 0 {
	//	// TODO: handle error of peer sending no blocks
	//	return nil
	//}
	//
	//blocks := make([]*primitives.Block, len(blockMessage.Blocks))
	//for i := range blockMessage.Blocks {
	//	b, err := primitives.BlockFromProto(blockMessage.Blocks[i])
	//	if err != nil {
	//		return err
	//	}
	//	blocks[i] = b
	//}
	//
	//for _, b := range blocks {
	//	err := s.handleReceivedBlock(b, peer, true)
	//	if err != nil {
	//		return err
	//	}
	//}

	//epochBlockChunks, err := splitIncomingBlocksIntoChunks(blocks, s.blockchain.config, s.blockchain.View.Index)
	//if err != nil {
	//	return err
	//}
	//
	//for _, chunk := range epochBlockChunks {
	//	epochState, found := s.blockchain.stateManager.GetStateForHash(chunk[0].BlockHeader.ParentRoot)
	//	if !found {
	//		return fmt.Errorf("could not find parent block at slot %d with parent root %s", chunk[0].BlockHeader.SlotNumber, chunk[0].BlockHeader.ParentRoot)
	//	}
	//
	//	tipNode := s.blockchain.View.Index.GetBlockNodeByHash(chunk[0].BlockHeader.ParentRoot)
	//
	//	epochStateCopy := epochState.Copy()
	//
	//	view := NewChainView(tipNode)
	//
	//	_, err := epochStateCopy.ProcessSlots(chunk[0].BlockHeader.SlotNumber, &view, s.blockchain.config)
	//	if err != nil {
	//		return err
	//	}
	//
	//	// proposer public keys
	//	var pubkeys []*bls.PublicKey
	//
	//	// attestation committee public keys
	//	var attestationAggregatedPubkeys []*bls.PublicKey
	//
	//	aggregateSig := bls.NewAggregateSignature()
	//	aggregateRandaoSig := bls.NewAggregateSignature()
	//	aggregateAttestationSig := bls.NewAggregateSignature()
	//
	//	blockHashes := make([][]byte, 0)
	//	randaoHashes := make([][]byte, 0)
	//	attestationHashes := make([][]byte, 0)
	//
	//	// go through each of the blocks in the past epoch or so
	//	for _, b := range chunk {
	//		_, err := epochStateCopy.ProcessSlots(b.BlockHeader.SlotNumber-1, &view, s.blockchain.config)
	//		if err != nil {
	//			return err
	//		}
	//
	//		proposerIndex, err := epochStateCopy.GetBeaconProposerIndex(b.BlockHeader.SlotNumber-1, s.blockchain.config)
	//		if err != nil {
	//			return err
	//		}
	//
	//		proposerPub, err := epochStateCopy.ValidatorRegistry[proposerIndex].GetPublicKey()
	//		if err != nil {
	//			return err
	//		}
	//
	//		// keep track of all proposer pubkeys for this epoch
	//		pubkeys = append(pubkeys, proposerPub)
	//
	//		blockSig, err := bls.DeserializeSignature(b.BlockHeader.Signature)
	//		if err != nil {
	//			return err
	//		}
	//
	//		randaoSig, err := bls.DeserializeSignature(b.BlockHeader.RandaoReveal)
	//		if err != nil {
	//			return err
	//		}
	//
	//		// aggregate both the block sig and the randao sig
	//		aggregateSig.AggregateSig(blockSig)
	//		aggregateRandaoSig.AggregateSig(randaoSig)
	//
	//		//
	//		blockWithoutSignature := b.Copy()
	//		blockWithoutSignature.BlockHeader.Signature = bls.EmptySignature.Serialize()
	//		blockWithoutSignatureRoot, err := ssz.HashTreeRoot(blockWithoutSignature)
	//		if err != nil {
	//			return err
	//		}
	//
	//		proposal := primitives.ProposalSignedData{
	//			Slot:      b.BlockHeader.SlotNumber,
	//			Shard:     s.blockchain.config.BeaconShardNumber,
	//			BlockHash: blockWithoutSignatureRoot,
	//		}
	//
	//		proposalRoot, err := ssz.HashTreeRoot(proposal)
	//		if err != nil {
	//			return err
	//		}
	//
	//		blockHashes = append(blockHashes, proposalRoot[:])
	//
	//		var slotBytes [8]byte
	//		binary.BigEndian.PutUint64(slotBytes[:], b.BlockHeader.SlotNumber)
	//		slotBytesHash := chainhash.HashH(slotBytes[:])
	//
	//		randaoHashes = append(randaoHashes, slotBytesHash[:])
	//
	//		for _, att := range b.BlockBody.Attestations {
	//			participants, err := epochStateCopy.GetAttestationParticipants(att.Data, att.ParticipationBitfield, s.blockchain.config)
	//			if err != nil {
	//				return err
	//			}
	//
	//			dataRoot, err := ssz.HashTreeRoot(primitives.AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
	//			if err != nil {
	//				return err
	//			}
	//
	//			groupPublicKey := bls.NewAggregatePublicKey()
	//			for _, p := range participants {
	//				pub, err := epochStateCopy.ValidatorRegistry[p].GetPublicKey()
	//				if err != nil {
	//					return err
	//				}
	//				groupPublicKey.AggregatePubKey(pub)
	//			}
	//
	//			aggSig, err := bls.DeserializeSignature(att.AggregateSig)
	//			if err != nil {
	//				return err
	//			}
	//
	//			aggregateAttestationSig.AggregateSig(aggSig)
	//
	//			attestationHashes = append(attestationHashes, dataRoot[:])
	//			attestationAggregatedPubkeys = append(attestationAggregatedPubkeys, groupPublicKey)
	//		}
	//	}
	//
	//	valid := bls.VerifyAggregate(pubkeys, blockHashes, aggregateSig, bls.DomainProposal)
	//	if !valid {
	//		return errors.New("blocks did not validate")
	//	}
	//
	//	valid = bls.VerifyAggregate(pubkeys, randaoHashes, aggregateRandaoSig, bls.DomainRandao)
	//	if !valid {
	//		return errors.New("block randaos did not validate")
	//	}
	//
	//	if len(attestationAggregatedPubkeys) > 0 {
	//		valid = bls.VerifyAggregate(attestationAggregatedPubkeys, attestationHashes, aggregateAttestationSig, bls.DomainAttestation)
	//		if !valid {
	//			fmt.Printf("chunk: ")
	//			for _, b := range chunk {
	//				fmt.Printf("%d ", b.BlockHeader.SlotNumber)
	//			}
	//			fmt.Printf("\n")
	//			return errors.New("block attestations did not validate")
	//		}
	//	}
	//
	//	for _, b := range chunk {
	//
	//	}
	//}

	//lastBlockHash, err := ssz.HashTreeRoot(blockMessage.Blocks[len(blockMessage.Blocks)-1])
	//if err != nil {
	//	return err
	//}
	//
	//if !bytes.Equal(lastBlockHash[:], blockMessage.LatestBlockHash) && !bytes.Equal(blockMessage.LatestBlockHash, zeroHash[:]) {
	//	logger.Infof("continuing sync to block %x", blockMessage.LatestBlockHash)
	//
	//	// request all blocks up to this block
	//	peer.SendMessage(&pb.GetBlockMessage{
	//		LocatorHashes: s.blockchain.View.Chain.GetChainLocator(),
	//		HashStop:      blockMessage.LatestBlockHash,
	//	})
	//}
	//
	//peer.ProcessingRequest = false

	return nil
}

// RegisterPostProcessHook registers a hook called after a block has been processed.
func (s *SyncManager) RegisterPostProcessHook(hook func(*primitives.Block, *primitives.State, []primitives.Receipt)) {
	s.postProcessHook = hook
}

func (s *SyncManager) handleReceivedBlock(block *primitives.Block, peerFrom peer.ID, verifySignature bool) error {
	blockHash, err := ssz.HashTreeRoot(block)
	if err != nil {
		return err
	}

	if s.blockchain.View.Index.GetBlockNodeByHash(block.BlockHeader.ParentRoot) == nil {
		logrus.WithFields(logrus.Fields{
			"hash":       chainhash.Hash(block.BlockHeader.ParentRoot),
			"slotTrying": block.BlockHeader.SlotNumber,
		}).Info("requesting parent block")

		// request all blocks up to this block
		err := s.protocol.SendMessage(peerFrom, &pb.GetBlocksMessage{
			LocatorHashes: s.blockchain.View.Chain.GetChainLocator(),
			HashStop:      blockHash[:],
		})
		if err != nil {
			return err
		}

	} else {
		logrus.WithField("slot", block.BlockHeader.SlotNumber).Debug("processing")
		receipts, newState, err := s.blockchain.ProcessBlock(block, true, verifySignature)
		if err != nil {
			return err
		}

		if s.postProcessHook != nil && newState != nil {
			s.postProcessHook(block, newState, receipts)
		}
	}

	return nil
}

// ListenForBlocks listens for new blocks over the pub-sub network
// being broadcast as a result of finding them.
func (s SyncManager) ListenForBlocks() error {
	_, err := s.hostNode.SubscribeMessage("block", func(data []byte, from peer.ID) {
		blockProto := new(pb.Block)

		err := proto.Unmarshal(data, blockProto)
		if err != nil {
			logrus.Error(err)
			return
		}

		block, err := primitives.BlockFromProto(blockProto)
		if err != nil {
			logrus.Error(err)
			return
		}

		err = s.handleReceivedBlock(block, from, true)
		if err != nil {
			logrus.Error(err)
			return
		}
	})
	if err != nil {
		return err
	}

	return nil
}

// ListenForAttestations listens for new attestations over the pub-sub network.
func (s SyncManager) ListenForAttestations() error {
	_, err := s.hostNode.SubscribeMessage("attestation", func(data []byte, from peer.ID) {
		attestationProto := new(pb.Attestation)

		err := proto.Unmarshal(data, attestationProto)
		if err != nil {
			logrus.Error(err)
			return
		}

		attestation, err := primitives.AttestationFromProto(attestationProto)
		if err != nil {
			logrus.Error(err)
			return
		}

		if s.mempool != nil {
			err = s.mempool.ProcessNewAttestation(*attestation)
			if err != nil {
				logrus.Error(err)
				return
			}
		}
	})
	if err != nil {
		return err
	}

	return nil
}

const beaconSyncProtocol = protocol.ID("/phore/beacon/sync/1.0.0")

// TODO: these should be configurable
const maxSyncPeers = 16
const minSyncPeers = 8

// registerHandlers registers handlers with the host node.
func (s SyncManager) registerHandlers() (*p2p.ProtocolHandler, error) {
	handler, err := s.hostNode.RegisterProtocolHandler(beaconSyncProtocol, maxSyncPeers, minSyncPeers)
	if err != nil {
		return nil, err
	}

	err = handler.RegisterHandler("pb.GetBlockMessage", s.onMessageGetBlock)
	if err != nil {
		return nil, err
	}

	err = handler.RegisterHandler("pb.BlockMessage", s.onMessageGetBlock)
	if err != nil {
		return nil, err
	}

	err = handler.RegisterHandler("pb.MempoolMessage", s.onMessageMempool)
	if err != nil {
		return nil, err
	}

	err = handler.RegisterHandler("pb.GetMempoolMessage", s.onMessageGetMempool)
	if err != nil {
		return nil, err
	}

	return handler, nil
}

// TryInitialSync tries to select a peer to sync with and
// start downloading blocks.
func (s SyncManager) TryInitialSync() {
	peers := s.hostNode.GetPeerList()
	if len(peers) == 0 {
		return
	}

	if !s.syncStarted {
		s.syncStarted = true

		bestPeer := peers[0]
		bestPeerOutbound := s.hostNode.GetPeerDirection(peers[0]) == network.DirOutbound

		for _, p := range peers[1:] {
			outbound := s.hostNode.GetPeerDirection(p) == network.DirOutbound

			if outbound && !bestPeerOutbound {
				bestPeer = p
				bestPeerOutbound = outbound
			}
		}

		getBlockMessage := &pb.GetBlocksMessage{
			LocatorHashes: s.blockchain.View.Chain.GetChainLocator(),
			HashStop:      zeroHash[:],
		}

		err := s.protocol.SendMessage(bestPeer, getBlockMessage)
		if err != nil {
			logrus.Error(err)
			return
		}

		if s.mempool != nil {
			attestationsMessage := s.mempool.AttestationMempool.GetMempoolSummary()

			err = s.protocol.SendMessage(bestPeer, &attestationsMessage)
		}
	}
}

func (s SyncManager) onMessageMempool(peer peer.ID, message proto.Message) error {
	if s.mempool == nil {
		return nil
	}

	mempoolMessage := message.(*pb.AttestationMempoolMessage)

	for _, attProto := range mempoolMessage.Attestations {
		a, err := primitives.AttestationFromProto(attProto)
		if err != nil {
			return err
		}

		err = s.mempool.ProcessNewAttestation(*a)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s SyncManager) onMessageGetMempool(peer peer.ID, message proto.Message) error {
	//if s.mempool == nil {
	//	return nil
	//}
	//
	//getMempoolMessage := message.(*pb.GetMempoolMessage)
	//
	//attestationsPeerDoesNotHave, err := s.mempool.AttestationMempool.GetMempoolDifference(getMempoolMessage)
	//if err != nil {
	//	return err
	//}
	//
	//attestationsProto := make([]*pb.Attestation, len(attestationsPeerDoesNotHave))
	//for i := range attestationsProto {
	//	attestationsProto[i] = attestationsPeerDoesNotHave[i].ToProto()
	//}
	//
	//peer.SendMessage(&pb.MempoolMessage{
	//	Attestations: attestationsProto,
	//})

	return nil
}
