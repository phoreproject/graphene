package beacon

import "github.com/phoreproject/graphene/primitives"

type blockNodeAndValidator struct {
	node      *BlockNode
	validator uint32
}

// UpdateChainHead updates the blockchain head if needed
func (b *Blockchain) UpdateChainHead() error {
	validators := b.View.justifiedHead.State.ValidatorRegistry
	activeValidatorIndices := primitives.GetActiveValidatorIndices(validators)
	var targets []blockNodeAndValidator
	for _, i := range activeValidatorIndices {
		bl, err := b.getLatestAttestationTarget(i)
		if err != nil {
			continue
		}
		targets = append(targets, blockNodeAndValidator{
			node:      bl,
			validator: i})
	}

	getVoteCount := func(block *BlockNode) uint64 {
		votes := uint64(0)
		for _, target := range targets {
			node := target.node.GetAncestorAtSlot(block.Slot)
			if node == nil {
				return 0
			}
			if node.Hash.IsEqual(&block.Hash) {
				votes += b.View.justifiedHead.State.GetEffectiveBalance(target.validator, b.config) / 1e8
			}
		}
		return votes
	}

	head, _ := b.View.GetJustifiedHead()

	// this may seem weird, but it occurs when importing when the justified block is not
	// imported, but the finalized head is. It should never occur other than that
	if head == nil {
		head, _ = b.View.GetFinalizedHead()
	}

	for {
		children := head.Children
		if len(children) == 0 {
			b.View.Chain.SetTip(head)

			err := b.DB.SetHeadBlock(head.Hash)
			if err != nil {
				return err
			}

			return nil
		}
		bestVoteCountChild := children[0]
		bestVotes := getVoteCount(bestVoteCountChild)
		for _, c := range children[1:] {
			vc := getVoteCount(c)
			if vc > bestVotes {
				bestVoteCountChild = c
				bestVotes = vc
			}
		}
		head = bestVoteCountChild
	}
}
