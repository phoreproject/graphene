package beacon

import (
	"github.com/phoreproject/synapse/chainhash"
	"github.com/sirupsen/logrus"
)

func (b *Blockchain) populateJustifiedAndFinalizedNodes() error {
	finalizedHead, err := b.DB.GetFinalizedHead()
	if err != nil {
		return err
	}

	finalizedHeadState, err := b.DB.GetFinalizedState()
	if err != nil {
		return err
	}

	justifiedHead, err := b.DB.GetJustifiedHead()
	if err != nil {
		return err
	}

	justifiedHeadState, err := b.DB.GetJustifiedState()
	if err != nil {
		return err
	}

	b.View.SetFinalizedHead(*finalizedHead, *finalizedHeadState)
	b.View.SetJustifiedHead(*justifiedHead, *justifiedHeadState)

	return nil
}

func (b *Blockchain) populateBlockIndexFromDatabase(genesisHash chainhash.Hash) error {
	finalizedHash, err := b.DB.GetFinalizedHead()
	if err != nil {
		return err
	}

	queue := []chainhash.Hash{genesisHash}

	for len(queue) > 0 {
		nodeToFind := queue[0]
		queue = queue[1:]

		nodeDisk, err := b.DB.GetBlockNode(nodeToFind)
		if err != nil {
			return err
		}

		_, err = b.View.Index.LoadBlockNode(nodeDisk)
		if err != nil {
			return err
		}

		if nodeToFind.IsEqual(finalizedHash) {
			// don't process any children of the finalized node (that happens when we load state)
			continue
		}

		logrus.WithField("processed", nodeDisk.Hash).Debug("imported block index")

		queue = append(queue, nodeDisk.Children...)
	}
	return nil
}

func (b *Blockchain) populateStateMap() error {
	// this won't have any children because we didn't add them while loading the block index
	finalizedNode := b.View.finalizedHead.BlockNode

	// get the children from the disk
	finalizedNodeWithChildren, err := b.DB.GetBlockNode(finalizedNode.Hash)
	if err != nil {
		return err
	}

	// queue the children to load
	loadQueue := finalizedNodeWithChildren.Children

	finalizedState, err := b.DB.GetFinalizedState()
	if err != nil {
		return err
	}

	err = b.stateManager.SetBlockState(finalizedNode.Hash, finalizedState)
	if err != nil {
		return err
	}

	b.View.Chain.SetTip(finalizedNode)

	err = b.stateManager.UpdateHead(finalizedNode.Hash)
	if err != nil {
		return err
	}

	for len(loadQueue) > 0 {
		itemToLoad := loadQueue[0]

		node, err := b.DB.GetBlockNode(itemToLoad)
		if err != nil {
			return err
		}

		//logrus.WithField("loading", node.Hash).WithField("slot", node.Slot).Debug("loading block from database")

		loadQueue = loadQueue[1:]

		bl, err := b.DB.GetBlockForHash(node.Hash)
		if err != nil {
			return err
		}

		_, _, err = b.AddBlockToStateMap(bl, false)
		if err != nil {
			logrus.Debug(err)
			return err
		}

		_, err = b.View.Index.LoadBlockNode(node)
		if err != nil {
			return err
		}

		// now, update the chain head
		err = b.UpdateChainHead()
		if err != nil {
			return err
		}

		loadQueue = append(loadQueue, node.Children...)
	}

	justifiedHead, err := b.DB.GetJustifiedHead()
	if err != nil {
		return err
	}

	justifiedHeadState, err := b.DB.GetJustifiedState()
	if err != nil {
		return err
	}

	b.View.SetJustifiedHead(*justifiedHead, *justifiedHeadState)

	// now, update the chain head
	err = b.UpdateChainHead()
	if err != nil {
		return err
	}

	return nil
}
