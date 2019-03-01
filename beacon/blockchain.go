package beacon

import (
	"errors"
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

func (bv *blockchainView) GetBlock(n uint64) (*blockNode, error) {
	bv.lock.Lock()
	defer bv.lock.Unlock()
	return getAncestor(bv.tip, n)
}

func (bv *blockchainView) Height() uint64 {
	bv.lock.Lock()
	defer bv.lock.Unlock()
	return bv.tip.height
}

// Blockchain represents a chain of blocks.
type Blockchain struct {
	chain        blockchainView
	db           db.Database
	config       *config.Config
	stateManager *StateManager
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
	// go up to the slot after the slot we're searching for
	for slot < current.slot {
		current = current.parent
	}
	return current, nil
}

// GetEpochBoundaryHash gets the hash of the parent block at the epoch boundary.
func (b *Blockchain) GetEpochBoundaryHash() (chainhash.Hash, error) {
	height := b.Height()
	epochBoundaryHeight := height - (height % b.config.EpochLength)
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
	}

	sm, err := NewStateManager(config, validators, genesisTime, skipValidation, b)
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
	node, err := b.addBlockNodeToIndex(&block0, blockHash)
	if err != nil {
		return nil, err
	}

	b.chain.finalizedHead = blockNodeAndState{node, initialState}
	b.chain.justifiedHead = blockNodeAndState{node, initialState}
	b.chain.tip = node

	err = b.stateManager.SetBlockState(blockHash, &initialState)
	if err != nil {
		return nil, err
	}

	if h, err := db.GetHeadBlock(); err == nil {
		headBlock, err := db.GetBlockForHash(*h)
		if err != nil {
			return nil, err
		}
		blocks := []*primitives.Block{headBlock}
		currentBlock := headBlock
		for !currentBlock.BlockHeader.ParentRoot.IsEqual(&zeroHash) {
			block, err := db.GetBlockForHash(currentBlock.BlockHeader.ParentRoot)
			if err != nil {
				return nil, err
			}
			if block.BlockHeader.SlotNumber != 0 {
				blocks = append(blocks, block)
			}
			currentBlock = block
		}

		for i := len(blocks) - 1; i >= 0; i-- {
			block := blocks[i]

			err := b.ProcessBlock(block)
			if err != nil {
				return nil, err
			}
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

	err = b.stateManager.SetBlockState(b.Tip(), newState)
	if err != nil {
		return err
	}

	b.stateManager.UpdateHead(b.Tip())

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

const (
	// RoleProposer is assigned to validators who need to propose a shard block.
	RoleProposer = iota

	// RoleAttester is assigned to validators who need to attest to a shard block.
	RoleAttester
)

const (
	// InitialForkVersion is version #1 of the chain.
	InitialForkVersion = iota
)

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
	targets := []blockNodeAndValidator{}
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

			err = b.db.SetHeadState(b.stateManager.GetHeadState())
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
	return b.chain.Height()
}

// LastBlock gets the last block in the chain
func (b *Blockchain) LastBlock() (*primitives.Block, error) {
	bl, err := b.chain.GetBlock(b.chain.Height())
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
