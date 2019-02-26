package beacon

import (
	"errors"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	logger "github.com/sirupsen/logrus"
)

// AddBlock adds a block header to the current chain. The block should already
// have been validated by this point.
func (b *Blockchain) AddBlock(block *primitives.Block) error {
	logger.Debug("adding block to cache and updating head if needed")

	err := b.db.SetBlock(*block)
	if err != nil {
		return err
	}

	return nil
}

func (b *Blockchain) processSlot(newState *primitives.State, previousBlockRoot chainhash.Hash) error {
	beaconProposerIndex, err := newState.GetBeaconProposerIndex(newState.Slot, newState.Slot, b.config)
	if err != nil {
		return err
	}

	// increase the slot number
	newState.Slot++

	// increase the randao skips of the proposer
	newState.ValidatorRegistry[beaconProposerIndex].ProposerSlots++

	newState.LatestBlockHashes[(newState.Slot-1)%b.config.LatestBlockRootsLength] = previousBlockRoot

	if newState.Slot%b.config.LatestBlockRootsLength == 0 {
		latestBlockHashesRoot, err := ssz.TreeHash(newState.LatestBlockHashes)
		if err != nil {
			return err
		}
		newState.BatchedBlockRoots = append(newState.BatchedBlockRoots, latestBlockHashesRoot)
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
func (b *Blockchain) ProcessBlock(block *primitives.Block) error {
	genesisTime := b.stateManager.GetGenesisTime()

	// VALIDATE BLOCK HERE
	if block.BlockHeader.SlotNumber*uint64(b.config.SlotDuration)+genesisTime > uint64(time.Now().Unix()) || block.BlockHeader.SlotNumber == 0 {
		return errors.New("block slot too soon")
	}

	blockHash, err := ssz.TreeHash(block)
	if err != nil {
		return err
	}

	blockHashStr := fmt.Sprintf("%x", blockHash)

	logger.WithField("hash", blockHashStr).Debug("processing new block")

	err = b.AddBlock(block)
	if err != nil {
		return err
	}

	logger.Debug("applying block")

	newState, err := b.stateManager.AddBlockToStateMap(block)
	if err != nil {
		return err
	}

	logger.Debug("applied with new state")

	node, err := b.addBlockNodeToIndex(block, blockHash)
	if err != nil {
		return err
	}

	logger.Debug("updating chain head")

	err = b.UpdateChainHead(block)
	if err != nil {
		return err
	}

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

	return nil
}

// GetNewRecentBlockHashes will take a list of recent block hashes and
// shift them to the right, filling them in with the parentHash provided.
func GetNewRecentBlockHashes(oldHashes []chainhash.Hash, parentSlot uint64, currentSlot uint64, parentHash chainhash.Hash) []chainhash.Hash {
	d := currentSlot - parentSlot
	newHashes := oldHashes[:]
	numberToAdd := int(d)
	if numberToAdd > len(oldHashes) {
		numberToAdd = len(oldHashes)
	}
	for i := 0; i < numberToAdd; i++ {
		newHashes = append(newHashes, parentHash)
	}
	return newHashes
}

// GetState gets a copy of the current state of the blockchain.
func (b *Blockchain) GetState() primitives.State {
	return b.stateManager.GetHeadState()
}
