package beacon

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	logger "github.com/sirupsen/logrus"
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
	postProcessHook func(*primitives.Block, *primitives.State, []primitives.Receipt)
}

// NewSyncManager creates a new sync manager
func NewSyncManager(hostNode *p2p.HostNode, blockchain *Blockchain) SyncManager {
	return SyncManager{
		hostNode:    hostNode,
		syncStarted: false,
		blockchain:  blockchain,
	}
}

// Connected returns whether the client should be considered connected to the
// network.
func (s SyncManager) Connected() bool {
	peers := s.hostNode.GetPeerList()

	for _, p := range peers {
		if p.IsConnected() {
			return true
		}
	}
	return false
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

	for currentBlockNode != nil && len(toSend) < limitBlocksToSend {
		currentBlock, err := s.blockchain.db.GetBlockForHash(currentBlockNode.Hash)
		if err != nil {
			return err
		}

		toSend = append(toSend, *currentBlock)

		if currentBlockNode.Hash.IsEqual(stopHash) {
			break
		}

		currentBlockNode = s.blockchain.View.Chain.Next(currentBlockNode)
	}

	if len(toSend) > 0 {
		logger.WithFields(logger.Fields{
			"from": firstCommonBlock.Slot,
			"to":   toSend[len(toSend)-1].BlockHeader.SlotNumber,
		}).Debug("sending blocks to peer")
	}

	blockMessage := &pb.BlockMessage{
		Blocks: make([]*pb.Block, len(toSend)),
	}

	for i := range toSend {
		blockMessage.Blocks[i] = toSend[i].ToProto()
	}

	peer.SendMessage(blockMessage)

	return nil
}

func (s SyncManager) onMessageBlock(peer *p2p.Peer, message proto.Message) error {
	// TODO: limits for this and only should receive blocks if requested

	blockMessage := message.(*pb.BlockMessage)

	logger.WithFields(logger.Fields{
		"from":   blockMessage.Blocks[0].Header.SlotNumber,
		"to":     blockMessage.Blocks[len(blockMessage.Blocks)-1].Header.SlotNumber,
		"number": len(blockMessage.Blocks),
	}).Debug("received blocks from sync")

	logger.Debug("checking signatures")

	peer.ProcessingRequest = true

	if len(blockMessage.Blocks) == 0 {
		// TODO: handle error of peer sending no blocks
		return nil
	}

	firstBlock, err := primitives.BlockFromProto(blockMessage.Blocks[0])
	if err != nil {
		return err
	}

	epochBlockChunks := make([][]*primitives.Block, 0)
	currentEpochChunk := make([]*primitives.Block, 0)
	currentEpoch := firstBlock.BlockHeader.SlotNumber / s.blockchain.config.EpochLength

	logger.WithFields(logger.Fields{
		"firstBlock": blockMessage.Blocks[0].Header.SlotNumber,
		"lastBlock":  blockMessage.Blocks[len(blockMessage.Blocks)-1].Header.SlotNumber,
	}).Info("received blocks from peer")

	var block *primitives.Block

	// TODO: separate the chunking code into own function and test it

	for i := range blockMessage.Blocks {
		block, err = primitives.BlockFromProto(blockMessage.Blocks[i])
		if err != nil {
			return err
		}

		blockHash, err := ssz.TreeHash(block)
		if err != nil {
			return err
		}

		if s.blockchain.View.Index.Has(blockHash) {
			// ignore blocks we already have.
			continue
		}

		addToNextEpoch := true

		if currentEpoch != block.BlockHeader.SlotNumber/s.blockchain.config.EpochLength || i == len(blockMessage.Blocks)-1 {
			if block.BlockHeader.SlotNumber%s.blockchain.config.EpochLength == 0 || i == len(blockMessage.Blocks)-1 {
				currentEpochChunk = append(currentEpochChunk, block)
				addToNextEpoch = false
			}
			if len(currentEpochChunk) > 0 {
				epochBlockChunks = append(epochBlockChunks, currentEpochChunk)
			}
			currentEpochChunk = make([]*primitives.Block, 0)
			currentEpoch = block.BlockHeader.SlotNumber / s.blockchain.config.EpochLength
		}

		// add to chunk if this is not the next epoch
		if addToNextEpoch {
			currentEpochChunk = append(currentEpochChunk, block)
		}
	}

	for _, chunk := range epochBlockChunks {
		epochState, found := s.blockchain.stateManager.GetStateForHash(chunk[0].BlockHeader.ParentRoot)
		if !found {
			return fmt.Errorf("could not find parent block at slot %d with parent root %s", chunk[0].BlockHeader.SlotNumber, chunk[0].BlockHeader.ParentRoot)
		}

		tipNode := s.blockchain.View.Index.GetBlockNodeByHash(chunk[0].BlockHeader.ParentRoot)

		epochStateCopy := epochState.Copy()

		view := NewChainView(tipNode, s.blockchain)

		err := epochStateCopy.ProcessSlots(chunk[0].BlockHeader.SlotNumber, &view, s.blockchain.config)
		if err != nil {
			return err
		}

		// proposer public keys
		var pubkeys []*bls.PublicKey

		// attestation committee public keys
		var attestationAggregatedPubkeys []*bls.PublicKey

		aggregateSig := bls.NewAggregateSignature()
		aggregateRandaoSig := bls.NewAggregateSignature()
		aggregateAttestationSig := bls.NewAggregateSignature()

		blockHashes := make([][]byte, 0)
		randaoHashes := make([][]byte, 0)
		attestationHashes := make([][]byte, 0)

		// go through each of the blocks in the past epoch or so
		for _, b := range chunk {
			err := epochStateCopy.ProcessSlots(b.BlockHeader.SlotNumber, &view, s.blockchain.config)
			if err != nil {
				return err
			}

			proposerIndex, err := epochStateCopy.GetBeaconProposerIndex(epochStateCopy.Slot-1, b.BlockHeader.SlotNumber-1, s.blockchain.config)
			if err != nil {
				return err
			}

			proposerPub, err := epochStateCopy.ValidatorRegistry[proposerIndex].GetPublicKey()
			if err != nil {
				return err
			}

			// keep track of all proposer pubkeys for this epoch
			pubkeys = append(pubkeys, proposerPub)

			blockSig, err := bls.DeserializeSignature(b.BlockHeader.Signature)
			if err != nil {
				return err
			}

			randaoSig, err := bls.DeserializeSignature(b.BlockHeader.RandaoReveal)
			if err != nil {
				return err
			}

			// aggregate both the block sig and the randao sig
			aggregateSig.AggregateSig(blockSig)
			aggregateRandaoSig.AggregateSig(randaoSig)

			//
			blockWithoutSignature := b.Copy()
			blockWithoutSignature.BlockHeader.Signature = bls.EmptySignature.Serialize()
			blockWithoutSignatureRoot, err := ssz.TreeHash(blockWithoutSignature)
			if err != nil {
				return err
			}

			proposal := primitives.ProposalSignedData{
				Slot:      b.BlockHeader.SlotNumber,
				Shard:     s.blockchain.config.BeaconShardNumber,
				BlockHash: blockWithoutSignatureRoot,
			}

			proposalRoot, err := ssz.TreeHash(proposal)
			if err != nil {
				return err
			}

			blockHashes = append(blockHashes, proposalRoot[:])

			var slotBytes [8]byte
			binary.BigEndian.PutUint64(slotBytes[:], b.BlockHeader.SlotNumber)
			slotBytesHash := chainhash.HashH(slotBytes[:])

			randaoHashes = append(randaoHashes, slotBytesHash[:])

			for _, att := range b.BlockBody.Attestations {
				participants, err := epochStateCopy.GetAttestationParticipants(att.Data, att.ParticipationBitfield, s.blockchain.config)
				if err != nil {
					return err
				}

				dataRoot, err := ssz.TreeHash(primitives.AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
				if err != nil {
					return err
				}

				groupPublicKey := bls.NewAggregatePublicKey()
				for _, p := range participants {
					pub, err := epochStateCopy.ValidatorRegistry[p].GetPublicKey()
					if err != nil {
						return err
					}
					groupPublicKey.AggregatePubKey(pub)
				}

				aggSig, err := bls.DeserializeSignature(att.AggregateSig)
				if err != nil {
					return err
				}

				aggregateAttestationSig.AggregateSig(aggSig)

				attestationHashes = append(attestationHashes, dataRoot[:])
				attestationAggregatedPubkeys = append(attestationAggregatedPubkeys, groupPublicKey)
			}
		}

		valid := bls.VerifyAggregate(pubkeys, blockHashes, aggregateSig, bls.DomainProposal)
		if !valid {
			return errors.New("blocks did not validate")
		}

		valid = bls.VerifyAggregate(pubkeys, randaoHashes, aggregateRandaoSig, bls.DomainRandao)
		if !valid {
			return errors.New("block randaos did not validate")
		}

		valid = bls.VerifyAggregate(attestationAggregatedPubkeys, attestationHashes, aggregateAttestationSig, bls.DomainAttestation)
		if !valid {
			return errors.New("block attestations did not validate")
		}

		for _, b := range chunk {
			err = s.handleReceivedBlock(b, peer, false)
			if err != nil {
				return err
			}
		}
	}

	peer.ProcessingRequest = false

	return nil
}

// RegisterPostProcessHook registers a hook called after a block has been processed.
func (s *SyncManager) RegisterPostProcessHook(hook func(*primitives.Block, *primitives.State, []primitives.Receipt)) {
	s.postProcessHook = hook
}

func (s SyncManager) handleReceivedBlock(block *primitives.Block, peerFrom *p2p.Peer, verifySignature bool) error {
	blockHash, err := ssz.TreeHash(block)
	if err != nil {
		return err
	}

	if s.blockchain.View.Index.GetBlockNodeByHash(block.BlockHeader.ParentRoot) == nil {
		// if we haven't handshaked with them, give up
		if peerFrom == nil {
			return nil
		}

		logger.WithFields(logger.Fields{
			"hash":       chainhash.Hash(block.BlockHeader.ParentRoot),
			"slotTrying": block.BlockHeader.SlotNumber,
		}).Info("requesting parent block")

		// request all blocks up to this block
		peerFrom.SendMessage(&pb.GetBlockMessage{
			LocatorHashes: s.blockchain.View.Chain.GetChainLocator(),
			HashStop:      blockHash[:],
		})

	} else {
		logger.WithField("slot", block.BlockHeader.SlotNumber).Debug("processing")
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
		peerFrom := s.hostNode.GetPeerByID(from)

		if peerFrom == nil {
			return
		}

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

		// if we're still syncing, ignore
		if peerFrom.ProcessingRequest {
			return
		}

		err = s.handleReceivedBlock(block, peerFrom, true)
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

		bestPeer.SendMessage(getBlockMessage)
	}
}
