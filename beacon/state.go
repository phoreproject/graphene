package beacon

import (
	"errors"
	"github.com/phoreproject/synapse/primitives/proofs"
	"time"

	"github.com/phoreproject/synapse/ssz"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/utils"
	logger "github.com/sirupsen/logrus"
)

// StoreBlock adds a block header to the current chain. The block should already
// have been validated by this point.
func (b *Blockchain) StoreBlock(block *primitives.Block) error {
	err := b.DB.SetBlock(*block)
	if err != nil {
		return err
	}

	return nil
}

// AddBlockToStateMap calculates the state after applying block and adds it
// to the state map.
func (b *Blockchain) AddBlockToStateMap(block *primitives.Block, verifySignature bool) (*primitives.EpochTransitionOutput, *primitives.State, error) {
	return b.stateManager.AddBlockToStateMap(block, verifySignature)
}

// ProcessBlock is called when a block is received from a peer.
func (b *Blockchain) ProcessBlock(block *primitives.Block, checkTime bool, verifySignature bool) (*primitives.EpochTransitionOutput, *primitives.State, error) {
	genesisTime := b.stateManager.GetGenesisTime()

	validationStart := time.Now()

	if checkTime && (block.BlockHeader.SlotNumber*uint64(b.config.SlotDuration)+genesisTime > uint64(utils.Now().Unix()) || block.BlockHeader.SlotNumber == 0) {
		return nil, nil, errors.New("block slot too soon")
	}

	seen := b.View.Index.Has(block.BlockHeader.ParentRoot)
	if !seen {
		return nil, nil, errors.New("do not have parent block")
	}

	blockHash, err := ssz.HashTreeRoot(block)
	if err != nil {
		return nil, nil, err
	}

	seen = b.View.Index.Has(blockHash)

	if seen {
		// we've already processed this block
		return nil, nil, nil
	}

	validationTime := time.Since(validationStart)

	logger.WithFields(logger.Fields{
		"hash": chainhash.Hash(blockHash),
		"slot": block.BlockHeader.SlotNumber,
	}).Info("processing new block")

	stateCalculationStart := time.Now()

	output, newState, err := b.AddBlockToStateMap(block, verifySignature)
	if err != nil {
		return nil, nil, err
	}

	b.notifeeLock.Lock()
	for _, n := range b.notifees {
		for _, c := range output.Crosslinks {
			n.CrosslinkCreated(&c.Crosslink, c.ShardID)
		}
	}
	b.notifeeLock.Unlock()

	reasons := map[uint8]int64{}

	netRewarded := int64(0)
	for _, r := range output.Receipts {
		netRewarded += r.Amount
		if _, found := reasons[r.Type]; found {
			reasons[r.Type] += r.Amount
		} else {
			reasons[r.Type] = r.Amount
		}
	}

	if len(reasons) > 0 {
		for reason, amount := range reasons {
			if amount > 0 {
				logger.Debugf("reward: %-50s %.08f PHR", primitives.ReceiptTypeToMeaning(reason), float64(amount)/1e8)
			} else {
				logger.Debugf("penalty: %-50s %.08f PHR", primitives.ReceiptTypeToMeaning(reason), float64(amount)/1e8)
			}
		}
		logger.Debugf("net rewards:                                               %.08f PHR", float64(netRewarded)/1e8)
	}

	stateCalculationTime := time.Since(stateCalculationStart)

	blockStorageStart := time.Now()

	err = b.StoreBlock(block)
	if err != nil {
		return nil, nil, err
	}

	blockStorageTime := time.Since(blockStorageStart)

	stateRoot, err := ssz.HashTreeRoot(newState)
	if err != nil {
		return nil, nil, err
	}

	//logger.Debug("applied with new state")

	node, err := b.View.Index.AddBlockNodeToIndex(block, blockHash, stateRoot, proofs.GetValidatorHash(newState))
	if err != nil {
		return nil, nil, err
	}

	databaseTipUpdateStart := time.Now()

	err = b.DB.TransactionalUpdate(func(transaction interface{}) error {
		// set the block node in the database
		err = b.DB.SetBlockNode(BlockNodeToDisk(*node), transaction)
		if err != nil {
			return err
		}

		// update the parent node in the database
		err = b.DB.SetBlockNode(BlockNodeToDisk(*node.Parent), transaction)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	databaseTipUpdateTime := time.Since(databaseTipUpdateStart)

	attestationUpdateStart := time.Now()

	for _, a := range block.BlockBody.Attestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, b.config)
		if err != nil {
			return nil, nil, err
		}

		err = b.DB.SetLatestAttestationsIfNeeded(participants, a)
		if err != nil {
			return nil, nil, err
		}
	}

	attestationUpdateEnd := time.Since(attestationUpdateStart)

	//logger.Debug("updating chain head")

	updateChainHeadStart := time.Now()

	err = b.UpdateChainHead()
	if err != nil {
		return nil, nil, err
	}

	updateChainHeadTime := time.Since(updateChainHeadStart)

	connectBlockSignalStart := time.Now()

	b.notifeeLock.Lock()
	for _, n := range b.notifees {
		go n.ConnectBlock(block)
	}
	b.notifeeLock.Unlock()

	connectBlockSignalTime := time.Since(connectBlockSignalStart)

	finalizedStateUpdateStart := time.Now()

	finalizedNode := node.GetAncestorAtSlot(newState.FinalizedEpoch * b.config.EpochLength)
	if finalizedNode == nil {
		return nil, nil, errors.New("could not find finalized node in block index")
	}
	finalizedState, found := b.stateManager.GetStateForHash(finalizedNode.Hash)
	if !found {
		return nil, nil, errors.New("could not find finalized block Hash in state map")
	}

	if !b.View.SetFinalizedHead(finalizedNode.Hash, *finalizedState) {
		return nil, nil, errors.New("could not set finalized head")
	}

	err = b.DB.SetFinalizedState(*finalizedState)
	if err != nil {
		return nil, nil, err
	}

	err = b.DB.SetFinalizedHead(finalizedNode.Hash)
	if err != nil {
		return nil, nil, err
	}

	justifiedNode := node.GetAncestorAtSlot(newState.JustifiedEpoch * b.config.EpochLength)
	if justifiedNode == nil {
		return nil, nil, errors.New("could not find justified node in block index")
	}

	justifiedState, found := b.stateManager.GetStateForHash(justifiedNode.Hash)
	if !found {
		return nil, nil, errors.New("could not find justified block Hash in state map")
	}
	if !b.View.SetJustifiedHead(justifiedNode.Hash, *justifiedState) {
		return nil, nil, errors.New("could not set justified head")
	}

	err = b.DB.SetJustifiedState(*justifiedState)
	if err != nil {
		return nil, nil, err
	}

	err = b.DB.SetJustifiedHead(justifiedNode.Hash)
	if err != nil {
		return nil, nil, err
	}

	finalizedStateUpdateTime := time.Since(finalizedStateUpdateStart)

	stateCleanupStart := time.Now()

	err = b.stateManager.DeleteStateBeforeFinalizedSlot(finalizedNode.Slot)
	if err != nil {
		return nil, nil, err
	}

	stateCleanupTime := time.Since(stateCleanupStart)

	logger.WithFields(logger.Fields{
		"validation":         validationTime,
		"stateCalculation":   stateCalculationTime,
		"storage":            blockStorageTime,
		"databaseTipUpdate":  databaseTipUpdateTime,
		"attestationUpdate":  attestationUpdateEnd,
		"updateChainHead":    updateChainHeadTime,
		"connectBlockSignal": connectBlockSignalTime,
		"finalizedUpdate":    finalizedStateUpdateTime,
		"stateCleanup":       stateCleanupTime,
		"totalTime":          time.Since(validationStart),
	})

	return output, newState, nil
}

// GetState gets a copy of the current state of the blockchain.
func (b *Blockchain) GetState() primitives.State {
	tipHash := b.View.Chain.Tip().Hash

	state, found := b.stateManager.GetStateForHash(tipHash)
	if !found {
		panic("don't have state for tip")
	}

	return *state
}
