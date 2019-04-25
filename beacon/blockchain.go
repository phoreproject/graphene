package beacon

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/primitives"

	"github.com/phoreproject/synapse/chainhash"
	utilsync "github.com/phoreproject/synapse/utils/sync"
)

var zeroHash = chainhash.Hash{}

type blockNode struct {
	hash     chainhash.Hash
	height   uint64
	slot     uint64
	parent   *blockNode
	children []*blockNode
}

type blockNodeAndState struct {
	*blockNode
	primitives.State
}

// blockchainView is the state of GHOST-LMD
type blockchainView struct {
	finalizedHead blockNodeAndState
	justifiedHead blockNodeAndState
	tip           *blockNode
	index         map[chainhash.Hash]*blockNode
	lock          *sync.Mutex
}

func (bv *blockchainView) getBlock(n uint64) (*blockNode, error) {
	bv.lock.Lock()
	defer bv.lock.Unlock()
	return getAncestor(bv.tip, n)
}

func (bv *blockchainView) height() uint64 {
	bv.lock.Lock()
	defer bv.lock.Unlock()
	return bv.tip.height
}

func (bv *blockchainView) seenBlock(blockHash chainhash.Hash) bool {
	bv.lock.Lock()
	_, found := bv.index[blockHash]
	bv.lock.Unlock()
	return found
}

// Blockchain represents a chain of blocks.
type Blockchain struct {
	chain        blockchainView
	db           db.Database
	config       *config.Config
	stateManager *StateManager

	ConnectBlockNotifier *utilsync.Signal
}

func blockNodeToHash(b *blockNode) chainhash.Hash {
	if b == nil {
		return chainhash.Hash{}
	}
	return b.hash
}

func blockNodeToDisk(b blockNode) db.BlockNodeDisk {
	childrenHashes := make([]chainhash.Hash, len(b.children))
	for i, c := range b.children {
		childrenHashes[i] = blockNodeToHash(c)
	}

	return db.BlockNodeDisk{
		Hash:     b.hash,
		Height:   b.height,
		Slot:     b.slot,
		Parent:   blockNodeToHash(b.parent),
		Children: childrenHashes,
	}
}

func (b *Blockchain) addBlockNodeToIndex(block *primitives.Block, blockHash chainhash.Hash) (*blockNode, error) {
	if node, found := b.chain.index[blockHash]; found {
		return node, nil
	}

	parentRoot := block.BlockHeader.ParentRoot
	parentNode, found := b.chain.index[parentRoot]
	if block.BlockHeader.SlotNumber == 0 {
		parentNode = nil
	} else if !found {
		return nil, errors.New("could not find parent block for incoming block")
	}
	height := uint64(0)
	if parentNode != nil {
		height = parentNode.height + 1
	}

	node := &blockNode{
		hash:     blockHash,
		height:   height,
		slot:     block.BlockHeader.SlotNumber,
		parent:   parentNode,
		children: []*blockNode{},
	}

	b.chain.index[blockHash] = node

	if parentNode != nil {
		parentNode.children = append(parentNode.children, node)
	}

	return node, nil
}

// getAncestor gets the ancestor of a block at a certain slot.
func getAncestor(node *blockNode, slot uint64) (*blockNode, error) {
	current := node

	//if node == nil || node.slot < slot {
	//	return nil, errors.New("no ancestors with this slot number")
	//}

	// go up to the slot after the slot we're searching for
	for slot < current.slot {
		current = current.parent
	}
	return current, nil
}

// GetEpochBoundaryHash gets the hash of the parent block at the epoch boundary.
func (b *Blockchain) GetEpochBoundaryHash(slot uint64) (chainhash.Hash, error) {
	epochBoundaryHeight := slot - (slot % b.config.EpochLength)
	return b.GetHashBySlot(epochBoundaryHeight)
}

func (b *Blockchain) getLatestAttestation(validator uint32) (*primitives.Attestation, error) {
	return b.db.GetLatestAttestation(validator)
}

func (b *Blockchain) getLatestAttestationTarget(validator uint32) (*blockNode, error) {
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
	node, found := b.chain.index[h]
	if !found {
		return nil, errors.New("couldn't find block attested to by validator in index")
	}
	return node, nil
}

// NewBlockchainWithInitialValidators creates a new blockchain with the specified
// initial validators.
func NewBlockchainWithInitialValidators(db db.Database, config *config.Config, validators []InitialValidatorEntry, skipValidation bool, genesisTime uint64) (*Blockchain, error) {
	b := &Blockchain{
		db:     db,
		config: config,
		chain: blockchainView{
			index: make(map[chainhash.Hash]*blockNode),
			lock:  &sync.Mutex{},
		},
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

	err = b.db.SetBlock(block0)
	if err != nil {
		return nil, err
	}

	_, err = b.db.GetBlockNode(blockHash)
	if err != nil {
		// this is a new database, so let's populate it with default values
		node, err := b.addBlockNodeToIndex(&block0, blockHash)
		if err != nil {
			return nil, err
		}
		b.chain.tip = node

		b.chain.finalizedHead = blockNodeAndState{node, initialState}
		b.chain.justifiedHead = blockNodeAndState{node, initialState}
		err = b.db.SetJustifiedHead(node.hash)
		if err != nil {
			return nil, err
		}
		err = b.db.SetJustifiedState(initialState)
		if err != nil {
			return nil, err
		}
		err = b.db.SetFinalizedHead(node.hash)
		if err != nil {
			return nil, err
		}
		err = b.db.SetFinalizedState(initialState)
		if err != nil {
			return nil, err
		}
		b.chain.tip = node

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

// UpdateStateIfNeeded updates the state to a certain slot.
func (b *Blockchain) UpdateStateIfNeeded(upTo uint64) error {
	newState, err := b.stateManager.ProcessSlots(upTo, b.Tip())
	if err != nil {
		return err
	}

	tip := b.Tip()

	err = b.stateManager.SetBlockState(tip, newState)
	if err != nil {
		return err
	}

	err = b.stateManager.UpdateHead(b.Tip())
	if err != nil {
		return err
	}

	return nil
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
	node      *blockNode
	validator uint32
}

// UpdateChainHead updates the blockchain head if needed
func (b *Blockchain) UpdateChainHead() error {
	b.chain.lock.Lock()
	defer b.chain.lock.Unlock()
	validators := b.chain.justifiedHead.State.ValidatorRegistry
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

	getVoteCount := func(block *blockNode) uint64 {
		votes := uint64(0)
		for _, target := range targets {
			node, err := getAncestor(target.node, block.slot)
			if err != nil {
				return 0
			}
			if node.hash.IsEqual(&block.hash) {
				votes += b.chain.justifiedHead.State.GetEffectiveBalance(target.validator, b.config) / 1e8
			}
		}
		return votes
	}

	head := b.chain.justifiedHead.blockNode
	for {
		children := head.children
		if len(children) == 0 {
			b.chain.tip = head
			err := b.stateManager.UpdateHead(head.hash)
			if err != nil {
				return err
			}

			err = b.db.SetHeadBlock(head.hash)
			if err != nil {
				return err
			}

			logger.WithFields(logrus.Fields{
				"height": head.height,
				"hash":   head.hash.String(),
				"slot":   b.stateManager.GetHeadSlot(),
			}).Debug("set tip")
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

// Tip returns the block at the tip of the chain.
func (b Blockchain) Tip() chainhash.Hash {
	b.chain.lock.Lock()
	tip := b.chain.tip.hash
	b.chain.lock.Unlock()
	return tip
}

// GetHashBySlot gets the block hash at a certain slot.
func (b Blockchain) GetHashBySlot(slot uint64) (chainhash.Hash, error) {
	b.chain.lock.Lock()
	node, err := getAncestor(b.chain.tip, slot)
	b.chain.lock.Unlock()
	if err != nil {
		return chainhash.Hash{}, err
	}
	return node.hash, nil
}

// Height returns the height of the chain.
func (b *Blockchain) Height() uint64 {
	return b.chain.height()
}

// LastBlock gets the last block in the chain
func (b *Blockchain) LastBlock() (*primitives.Block, error) {
	bl, err := b.chain.getBlock(b.chain.height())
	if err != nil {
		return nil, err
	}
	block, err := b.db.GetBlockForHash(bl.hash)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GetBlockByHash gets a block by hash.
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

	finalizedBlockNode, found := b.chain.index[*finalizedHead]
	if !found {
		return fmt.Errorf("could not find finalized block node in block index (hash: %s)", finalizedHead)
	}

	b.chain.finalizedHead = blockNodeAndState{finalizedBlockNode, *finalizedHeadState}

	justifiedHead, err := b.db.GetJustifiedHead()
	if err != nil {
		return err
	}

	justifiedHeadState, err := b.db.GetJustifiedState()
	if err != nil {
		return err
	}

	justifiedBlockNode, found := b.chain.index[*finalizedHead]
	if !found {
		return fmt.Errorf("could not find justified block node in block index (hash: %s)", justifiedHead)
	}

	b.chain.justifiedHead = blockNodeAndState{justifiedBlockNode, *justifiedHeadState}

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

		node, err := b.db.GetBlockNode(nodeToFind)
		if err != nil {
			return err
		}

		parent := b.chain.index[node.Parent]

		newNode := &blockNode{
			hash:     node.Hash,
			height:   node.Height,
			slot:     node.Slot,
			parent:   parent,
			children: make([]*blockNode, 0),
		}

		b.chain.index[node.Hash] = newNode

		if nodeToFind.IsEqual(finalizedHash) {
			// don't process any children of the finalized node (that happens when we load state)
			continue
		}

		if parent != nil {
			parent.children = append(parent.children, newNode)
		}

		queue = append(queue, node.Children...)
	}
	return nil
}

func (b *Blockchain) populateStateMap() error {
	// this won't have any children because we didn't add them while loading the block index
	finalizedNode := b.chain.finalizedHead.blockNode

	// get the children from the disk
	finalizedNodeWithChildren, err := b.db.GetBlockNode(finalizedNode.hash)
	if err != nil {
		return err
	}

	// queue the children to load
	loadQueue := finalizedNodeWithChildren.Children

	finalizedState, err := b.db.GetFinalizedState()
	if err != nil {
		return err
	}

	err = b.stateManager.SetBlockState(finalizedNode.hash, finalizedState)
	if err != nil {
		return err
	}

	b.chain.tip = finalizedNode

	b.stateManager.UpdateHead(finalizedNode.hash)

	for len(loadQueue) > 0 {
		itemToLoad := loadQueue[0]

		node, err := b.db.GetBlockNode(itemToLoad)
		if err != nil {
			return err
		}

		logrus.WithField("loading", node.Hash).WithField("slot", node.Slot).Debug("loading block from database")

		loadQueue = loadQueue[1:]

		bl, err := b.db.GetBlockForHash(node.Hash)
		if err != nil {
			return err
		}

		_, err = b.AddBlockToStateMap(bl)
		if err != nil {
			logrus.Debug(err)
			return err
		}

		parent := b.chain.index[node.Parent]

		newNode := &blockNode{
			hash:     node.Hash,
			height:   node.Height,
			slot:     node.Slot,
			parent:   parent,
			children: make([]*blockNode, 0),
		}

		b.chain.index[node.Hash] = newNode

		if parent != nil {
			parent.children = append(parent.children, newNode)
		}

		// now, update the chain head
		err = b.UpdateChainHead()
		if err != nil {
			return err
		}

		loadQueue = append(loadQueue, node.Children...)
	}

	return nil
}

// GetBlockHashesAfterBlock gets all block hashes from the specified block to the tip. Returns an error if the specified
// block is not in the current chain.
func (b *Blockchain) GetBlockHashesAfterBlock(blockFrom chainhash.Hash) ([]chainhash.Hash, error) {
	current := *b.chain.tip
	count := 0

	// first make sure the block they are requesting is in our current chain
	for current.slot > 0 && !current.hash.IsEqual(&blockFrom) {
		current = *current.parent
		count++
	}
	if current.slot == 0 && !current.hash.IsEqual(&blockFrom) {
		return nil, errors.New("block is not in current chain")
	}

	current = *b.chain.tip // go back to the tip

	blockHashes := make([]chainhash.Hash, count)
	for !current.hash.IsEqual(&blockFrom) {
		blockHashes[uint64(len(blockHashes))+current.height-b.chain.tip.height-1] = current.hash
		current = *current.parent
	}

	return blockHashes, nil
}

// GetCurrentSlot gets the current slot according to the time.
func (b *Blockchain) GetCurrentSlot() uint64 {
	currentTime := uint64(time.Now().Unix())

	timeSinceGenesis := currentTime - b.stateManager.GetGenesisTime()

	return timeSinceGenesis / uint64(b.config.SlotDuration)
}
