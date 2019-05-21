package beacon

import (
	"errors"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/chainhash"
	utilsync "github.com/phoreproject/synapse/utils/sync"
)

var zeroHash = chainhash.Hash{}

// Blockchain represents a chain of blocks.
type Blockchain struct {
	View         *BlockchainView
	db           db.Database
	config       *config.Config
	stateManager *StateManager

	ConnectBlockNotifier *utilsync.Signal
}

func blockNodeToHash(b *BlockNode) chainhash.Hash {
	if b == nil {
		return chainhash.Hash{}
	}
	return b.Hash
}

func blockNodeToDisk(b BlockNode) db.BlockNodeDisk {
	childrenHashes := make([]chainhash.Hash, len(b.Children))
	for i, c := range b.Children {
		childrenHashes[i] = blockNodeToHash(c)
	}

	return db.BlockNodeDisk{
		Hash:     b.Hash,
		Height:   b.Height,
		Slot:     b.Slot,
		Parent:   blockNodeToHash(b.Parent),
		Children: childrenHashes,
	}
}

// GetEpochBoundaryHash gets the Hash of the parent block at the epoch boundary.
func (b *Blockchain) GetEpochBoundaryHash(slot uint64) (chainhash.Hash, error) {
	epochBoundaryHeight := slot - (slot % b.config.EpochLength)
	return b.GetHashBySlot(epochBoundaryHeight)
}

func (b *Blockchain) getLatestAttestation(validator uint32) (*primitives.Attestation, error) {
	return b.db.GetLatestAttestation(validator)
}

func (b *Blockchain) getLatestAttestationTarget(validator uint32) (*BlockNode, error) {
	att, err := b.db.GetLatestAttestation(validator)
	if err != nil {
		return nil, err
	}
	bl, err := b.db.GetBlockForHash(att.Data.BeaconBlockHash)
	if err != nil {
		return nil, err
	}
	h, err := ssz.TreeHash(bl)
	if err != nil {
		return nil, err
	}

	node := b.View.Index.GetBlockNodeByHash(h)
	if node == nil {
		return nil, errors.New("couldn't find block attested to by validator in index")
	}
	return node, nil
}

// NewBlockchainWithInitialValidators creates a new blockchain with the specified
// initial validators.
func NewBlockchainWithInitialValidators(db db.Database, config *config.Config, validators []InitialValidatorEntry, skipValidation bool, genesisTime uint64) (*Blockchain, error) {
	b := &Blockchain{
		db:                   db,
		config:               config,
		View:                 NewBlockchainView(),
		ConnectBlockNotifier: utilsync.NewSignal(),
	}

	sm, err := NewStateManager(config, validators, genesisTime, skipValidation, b, db)
	if err != nil {
		return nil, err
	}

	b.stateManager = sm

	initialState := sm.GetHeadState()

	stateRoot, err := ssz.TreeHash(initialState)
	if err != nil {
		return nil, err
	}

	block0 := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   0,
			StateRoot:    stateRoot,
			ParentRoot:   zeroHash,
			RandaoReveal: bls.EmptySignature.Serialize(),
			Signature:    bls.EmptySignature.Serialize(),
		},
		BlockBody: primitives.BlockBody{
			ProposerSlashings: []primitives.ProposerSlashing{},
			CasperSlashings:   []primitives.CasperSlashing{},
			Attestations:      []primitives.Attestation{},
			Deposits:          []primitives.Deposit{},
			Exits:             []primitives.Exit{},
		},
	}

	blockHash, err := ssz.TreeHash(block0)
	if err != nil {
		return nil, err
	}

	logrus.WithField("genesisHash", chainhash.Hash(blockHash)).Info("initializing blockchain with genesis block")

	err = b.db.SetBlock(block0)
	if err != nil {
		return nil, err
	}

	// this is a new database, so let's populate it with default values
	node, err := b.View.Index.AddBlockNodeToIndex(&block0, blockHash)
	if err != nil {
		return nil, err
	}

	_, err = b.db.GetBlockNode(blockHash)
	if err != nil {
		b.View.Chain.SetTip(node)

		b.View.SetFinalizedHead(blockHash, initialState)
		b.View.SetJustifiedHead(blockHash, initialState)
		err = b.db.SetJustifiedHead(node.Hash)
		if err != nil {
			return nil, err
		}
		err = b.db.SetJustifiedState(initialState)
		if err != nil {
			return nil, err
		}
		err = b.db.SetFinalizedHead(node.Hash)
		if err != nil {
			return nil, err
		}
		err = b.db.SetFinalizedState(initialState)
		if err != nil {
			return nil, err
		}
		b.View.Chain.SetTip(node)

		err = b.stateManager.SetBlockState(blockHash, &initialState)
		if err != nil {
			return nil, err
		}

		err = b.db.SetHeadBlock(blockHash)
		if err != nil {
			return nil, err
		}
	} else {
		logrus.Info("loading block index...")
		err := b.populateBlockIndexFromDatabase(blockHash)
		if err != nil {
			return nil, err
		}

		logrus.Info("loading justified and finalized states...")
		err = b.populateJustifiedAndFinalizedNodes()
		if err != nil {
			return nil, err
		}

		logrus.Info("populating state map...")
		err = b.populateStateMap()
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

// GetUpdatedState gets the tip, but with processed slots/epochs as appropriate.
func (b *Blockchain) GetUpdatedState(upTo uint64) (*primitives.State, error) {
	tip := b.View.Chain.Tip()

	tipState := b.stateManager.GetHeadState()

	tipStateCopy := tipState.Copy()

	view := NewChainView(tip, b)

	err := tipStateCopy.ProcessSlots(upTo, &view, b.config)
	if err != nil {
		return nil, err
	}

	return &tipStateCopy, nil
}

// GetNextSlotTime returns the timestamp of the next slot.
func (b *Blockchain) GetNextSlotTime() time.Time {
	return time.Unix(int64((b.stateManager.GetHeadSlot()+1)*uint64(b.config.SlotDuration)+b.stateManager.GetGenesisTime()), 0)
}

// InitialValidatorEntry is the validator entry to be added
// at the beginning of a blockchain.
type InitialValidatorEntry struct {
	PubKey                [96]byte
	ProofOfPossession     [48]byte
	WithdrawalShard       uint32
	WithdrawalCredentials chainhash.Hash
	DepositSize           uint64
}

// InitialValidatorEntryAndPrivateKey is an initial validator entry and private key.
type InitialValidatorEntryAndPrivateKey struct {
	Entry InitialValidatorEntry

	PrivateKey [32]byte
}

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
	// imported, but the finalized head is. It should never occurother than that
	if head == nil {
		head, _ = b.View.GetFinalizedHead()
	}

	for {
		children := head.Children
		if len(children) == 0 {
			b.View.Chain.SetTip(head)
			err := b.stateManager.UpdateHead(head.Hash)
			if err != nil {
				return err
			}

			err = b.db.SetHeadBlock(head.Hash)
			if err != nil {
				return err
			}

			//logger.WithFields(logrus.Fields{
			//	"height": head.Height,
			//	"hash":   head.Hash.String(),
			//	"slot":   b.stateManager.GetHeadSlot(),
			//}).Debug("set tip")
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

// GetHashBySlot gets the block Hash at a certain slot.
func (b Blockchain) GetHashBySlot(slot uint64) (chainhash.Hash, error) {
	tip := b.View.Chain.Tip()
	if tip.Slot < slot {
		return tip.Hash, nil
	}
	node := tip.GetAncestorAtSlot(slot)
	if node == nil {
		return chainhash.Hash{}, fmt.Errorf("no block at slot %d", slot)
	}
	return node.Hash, nil
}

// Height returns the height of the chain.
func (b *Blockchain) Height() uint64 {
	h := b.View.Chain.Height()
	if h < 0 {
		return 0
	}
	return uint64(h)
}

// LastBlock gets the last block in the chain
func (b *Blockchain) LastBlock() (*primitives.Block, error) {
	bl := b.View.Chain.Tip()
	block, err := b.db.GetBlockForHash(bl.Hash)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GetBlockByHash gets a block by Hash.
func (b *Blockchain) GetBlockByHash(h chainhash.Hash) (*primitives.Block, error) {
	block, err := b.db.GetBlockForHash(h)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GetConfig returns the config used by this blockchain
func (b *Blockchain) GetConfig() *config.Config {
	return b.config
}

func (b *Blockchain) populateJustifiedAndFinalizedNodes() error {
	finalizedHead, err := b.db.GetFinalizedHead()
	if err != nil {
		return err
	}

	finalizedHeadState, err := b.db.GetFinalizedState()
	if err != nil {
		return err
	}

	justifiedHead, err := b.db.GetJustifiedHead()
	if err != nil {
		return err
	}

	justifiedHeadState, err := b.db.GetJustifiedState()
	if err != nil {
		return err
	}

	b.View.SetFinalizedHead(*finalizedHead, *finalizedHeadState)
	b.View.SetJustifiedHead(*justifiedHead, *justifiedHeadState)

	return nil
}

func (b *Blockchain) populateBlockIndexFromDatabase(genesisHash chainhash.Hash) error {
	finalizedHash, err := b.db.GetFinalizedHead()
	if err != nil {
		return err
	}

	queue := []chainhash.Hash{genesisHash}

	for len(queue) > 0 {
		nodeToFind := queue[0]
		queue = queue[1:]

		nodeDisk, err := b.db.GetBlockNode(nodeToFind)
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
	finalizedNodeWithChildren, err := b.db.GetBlockNode(finalizedNode.Hash)
	if err != nil {
		return err
	}

	// queue the children to load
	loadQueue := finalizedNodeWithChildren.Children

	finalizedState, err := b.db.GetFinalizedState()
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

		node, err := b.db.GetBlockNode(itemToLoad)
		if err != nil {
			return err
		}

		//logrus.WithField("loading", node.Hash).WithField("slot", node.Slot).Debug("loading block from database")

		loadQueue = loadQueue[1:]

		bl, err := b.db.GetBlockForHash(node.Hash)
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

	justifiedHead, err := b.db.GetJustifiedHead()
	if err != nil {
		return err
	}

	justifiedHeadState, err := b.db.GetJustifiedState()
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

// GetCurrentSlot gets the current slot according to the time.
func (b *Blockchain) GetCurrentSlot() uint64 {
	currentTime := uint64(time.Now().Unix())

	timeSinceGenesis := currentTime - b.stateManager.GetGenesisTime()

	return timeSinceGenesis / uint64(b.config.SlotDuration)
}

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
	h, err := c.blockchain.GetHashBySlot(slot)
	if err != nil {
		return nil, err
	}
	state, found := c.blockchain.stateManager.GetStateForHash(h)
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

// GenesisHash gets the genesis hash for the chain.
func (b *Blockchain) GenesisHash() chainhash.Hash {
	h, _ := b.GetHashBySlot(0)
	return h
}
