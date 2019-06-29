package beacon

import (
	"errors"
	"fmt"

	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/sirupsen/logrus"
)

func blockNodeToHash(b *BlockNode) chainhash.Hash {
	if b == nil {
		return chainhash.Hash{}
	}
	return b.Hash
}

func blockNodeToDisk(b BlockNode) db.BlockNodeDisk {
	children := make([]chainhash.Hash, len(b.Children))
	for i := range children {
		children[i] = blockNodeToHash(b.Children[i])
	}

	return db.BlockNodeDisk{
		Hash:      b.Hash,
		Height:    b.Height,
		Slot:      b.Slot,
		StateRoot: b.StateRoot,
		Parent:    blockNodeToHash(b.Parent),
		Children:  children,
	}
}

// loadBlockchainFromDisk loads all information of the blockchain from disk
// into memory.
func (b *Blockchain) loadBlockchainFromDisk(genesisHash chainhash.Hash) error {
	logrus.Info("loading block index...")
	err := b.populateBlockIndexFromDatabase(genesisHash)
	if err != nil {
		return err
	}

	logrus.Info("loading justified and finalized states...")
	err = b.populateJustifiedAndFinalizedNodes()
	if err != nil {
		return err
	}

	logrus.Info("populating state map...")
	err = b.populateStateMap()
	if err != nil {
		return err
	}

	return nil
}

// initializeDatabase sets up the database initially with default
// values based on the genesis block.
func (b *Blockchain) initializeDatabase(node *BlockNode, initialState primitives.State) error {
	b.View.Chain.SetTip(node)

	b.View.SetFinalizedHead(node.Hash, initialState)
	b.View.SetJustifiedHead(node.Hash, initialState)
	err := b.DB.SetJustifiedHead(node.Hash)
	if err != nil {
		return err
	}
	err = b.DB.SetJustifiedState(initialState)
	if err != nil {
		return err
	}
	err = b.DB.SetFinalizedHead(node.Hash)
	if err != nil {
		return err
	}
	err = b.DB.SetFinalizedState(initialState)
	if err != nil {
		return err
	}
	b.View.Chain.SetTip(node)

	err = b.stateManager.SetBlockState(node.Hash, &initialState)
	if err != nil {
		return err
	}

	err = b.DB.SetHeadBlock(node.Hash)
	if err != nil {
		return err
	}

	return nil
}

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

	if !b.View.SetFinalizedHead(*finalizedHead, *finalizedHeadState) {
		return errors.New("could not find finalized head in index")
	}
	if !b.View.SetJustifiedHead(*justifiedHead, *justifiedHeadState) {
		fmt.Println(*justifiedHead)
		return errors.New("could not find justified head in index")
	}

	return nil
}

func (b *Blockchain) populateBlockIndexFromDatabase(genesisHash chainhash.Hash) error {
	justfiedHead, err := b.DB.GetJustifiedHead()
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

		fmt.Println(nodeDisk.Hash, justfiedHead)

		_, err = b.View.Index.LoadBlockNode(nodeDisk)
		if err != nil {
			return err
		}

		if nodeToFind.IsEqual(justfiedHead) {
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
