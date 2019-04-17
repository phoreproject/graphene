package beacon

import (
	"errors"
	"fmt"
	"time"

	"github.com/phoreproject/prysm/shared/ssz"

	"github.com/phoreproject/synapse/primitives"
	logger "github.com/sirupsen/logrus"
)

// StoreBlock adds a block header to the current chain. The block should already
// have been validated by this point.
func (b *Blockchain) StoreBlock(block *primitives.Block) error {
	err := b.db.SetBlock(*block)
	if err != nil {
		return err
	}

	return nil
}

func intSqrt(n uint64) uint64 {
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// ProcessBlock is called when a block is received from a peer.
func (b *Blockchain) ProcessBlock(block *primitives.Block, checkTime bool) error {
	genesisTime := b.stateManager.GetGenesisTime()

	validationStart := time.Now()

	// VALIDATE BLOCK HERE
	if checkTime && block.BlockHeader.SlotNumber*uint64(b.config.SlotDuration)+genesisTime > uint64(time.Now().Unix()) || block.BlockHeader.SlotNumber == 0 {
		return errors.New("block slot too soon")
	}

	seen := b.chain.seenBlock(block.BlockHeader.ParentRoot)
	if !seen {
		return errors.New("do not have parent block")
	}

	blockHash, err := ssz.TreeHash(block)
	if err != nil {
		return err
	}

	seen = b.chain.seenBlock(blockHash)
	if seen {
		// we've already processed this block
		return nil
	}

	validationTime := time.Since(validationStart)

	blockHashStr := fmt.Sprintf("%x", blockHash)

	logger.WithField("hash", blockHashStr).Info("processing new block")

	stateCalculationStart := time.Now()

	newState, err := b.stateManager.AddBlockToStateMap(block)
	if err != nil {
		return err
	}

	stateCalculationTime := time.Since(stateCalculationStart)

	blockStorageStart := time.Now()

	err = b.StoreBlock(block)
	if err != nil {
		return err
	}

	blockStorageTime := time.Since(blockStorageStart)

	logger.Debug("applied with new state")

	node, err := b.addBlockNodeToIndex(block, blockHash)
	if err != nil {
		return err
	}

	// TODO: these two database operations should be in a single transaction

	databaseTipUpdateStart := time.Now()

	// set the block node in the database
	err = b.db.SetBlockNode(blockNodeToDisk(*node))
	if err != nil {
		return err
	}

	// update the parent node in the database
	err = b.db.SetBlockNode(blockNodeToDisk(*node.parent))
	if err != nil {
		return err
	}

	databaseTipUpdateTime := time.Since(databaseTipUpdateStart)

	attestationUpdateStart := time.Now()

	for _, a := range block.BlockBody.Attestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, b.config)
		if err != nil {
			return err
		}

		err = b.db.SetLatestAttestationsIfNeeded(participants, a)
		if err != nil {
			return err
		}
	}

	attestationUpdateEnd := time.Since(attestationUpdateStart)

	logger.Debug("updating chain head")

	updateChainHeadStart := time.Now()

	err = b.UpdateChainHead()
	if err != nil {
		return err
	}

	updateChainHeadTime := time.Since(updateChainHeadStart)

	connectBlockSignalStart := time.Now()

	b.ConnectBlockNotifier.Signal(*block)

	connectBlockSignalTime := time.Since(connectBlockSignalStart)

	finalizedStateUpdateStart := time.Now()

	finalizedNode, err := getAncestor(node, newState.FinalizedSlot)
	if err != nil {
		return err
	}
	finalizedState, found := b.stateManager.GetStateForHash(finalizedNode.hash)
	if !found {
		return errors.New("could not find finalized block hash in state map")
	}

	finalizedNodeAndState := blockNodeAndState{finalizedNode, *finalizedState}
	b.chain.finalizedHead = finalizedNodeAndState

	err = b.db.SetFinalizedHead(finalizedNode.hash)
	if err != nil {
		return err
	}

	justifiedNode, err := getAncestor(node, newState.JustifiedSlot)
	if err != nil {
		return err
	}

	justifiedState, found := b.stateManager.GetStateForHash(justifiedNode.hash)
	if !found {
		return errors.New("could not find justified block hash in state map")
	}
	justifiedNodeAndState := blockNodeAndState{justifiedNode, *justifiedState}
	b.chain.justifiedHead = justifiedNodeAndState

	err = b.db.SetJustifiedHead(justifiedNode.hash)
	if err != nil {
		return err
	}

	finalizedStateUpdateTime := time.Since(finalizedStateUpdateStart)

	stateCleanupStart := time.Now()

	err = b.stateManager.DeleteStateBeforeFinalizedSlot(finalizedNode.slot)
	if err != nil {
		return err
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
	}).Debug("performance report for processing")

	return nil
}

// GetState gets a copy of the current state of the blockchain.
func (b *Blockchain) GetState() primitives.State {
	return b.stateManager.GetHeadState()
}
