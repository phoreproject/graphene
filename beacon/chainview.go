package beacon

import (
	"errors"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// ChainView is a view of a certain chain in the block tree so that block processing can access valid blocks.
type ChainView struct {
	tip *BlockNode

	// effectiveTipSlot is used when the chain is being updated (excluding blocks)
	effectiveTipSlot uint64

	blockchain *Blockchain
}

// NewChainView creates a new chain view with a certain tip
func NewChainView(tip *BlockNode, blockchain *Blockchain) ChainView {
	return ChainView{tip, tip.Slot, blockchain}
}

// SetTipSlot sets the effective tip slot (which may be updated due to slot transitions)
func (c *ChainView) SetTipSlot(slot uint64) {
	c.effectiveTipSlot = slot
}

// GetHashBySlot gets a hash of a block in a certain slot.
func (c *ChainView) GetHashBySlot(slot uint64) (chainhash.Hash, error) {
	ancestor := c.tip.GetAncestorAtSlot(slot)
	if ancestor == nil {
		if slot > c.effectiveTipSlot {
			return chainhash.Hash{}, errors.New("could not get block past tip")
		}
		ancestor = c.tip
	}
	return ancestor.Hash, nil
}

// Tip gets the tip of the blockchain.
func (c *ChainView) Tip() (chainhash.Hash, error) {
	return c.tip.Hash, nil
}

// GetStateBySlot gets a state in a certain slot
func (c *ChainView) GetStateBySlot(slot uint64) (*primitives.State, error) {
	n, err := c.GetHashBySlot(slot)
	if err != nil {
		return nil, err
	}
	state, found := c.blockchain.stateManager.GetStateForHash(n)
	if !found {
		return nil, errors.New("could not find state at slot")
	}

	return state, nil
}

var _ primitives.BlockView = (*ChainView)(nil)

// GetSubView gets a view of the blockchain at a certain tip.
func (b *Blockchain) GetSubView(tip chainhash.Hash) (ChainView, error) {
	tipNode := b.View.Index.GetBlockNodeByHash(tip)
	if tipNode == nil {
		return ChainView{}, errors.New("could not find tip node")
	}
	return NewChainView(tipNode, b), nil
}