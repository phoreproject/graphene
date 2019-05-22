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
)

var zeroHash = chainhash.Hash{}

// Blockchain represents a chain of blocks.
type Blockchain struct {
	View         *BlockchainView
	DB           db.Database
	config       *config.Config
	stateManager *StateManager

	Notifees []BlockchainNotifee
}

// BlockchainNotifee is a blockchain notifee.
type BlockchainNotifee interface {
	ConnectBlock(*primitives.Block)
}

// RegisterNotifee registers a notifee for blockchain
func (b *Blockchain) RegisterNotifee(n BlockchainNotifee) {
	b.Notifees = append(b.Notifees, n)
}

func blockNodeToHash(b *BlockNode) chainhash.Hash {
	if b == nil {
		return chainhash.Hash{}
	}
	return b.Hash
}

func blockNodeToDisk(b BlockNode) db.BlockNodeDisk {
	return db.BlockNodeDisk{
		Hash:   b.Hash,
		Height: b.Height,
		Slot:   b.Slot,
		Parent: blockNodeToHash(b.Parent),
	}
}

// GetEpochBoundaryHash gets the Hash of the parent block at the epoch boundary.
func (b *Blockchain) GetEpochBoundaryHash(slot uint64) (chainhash.Hash, error) {
	epochBoundaryHeight := slot - (slot % b.config.EpochLength)
	return b.GetHashBySlot(epochBoundaryHeight)
}

func (b *Blockchain) getLatestAttestation(validator uint32) (*primitives.Attestation, error) {
	return b.DB.GetLatestAttestation(validator)
}

func (b *Blockchain) getLatestAttestationTarget(validator uint32) (*BlockNode, error) {
	att, err := b.DB.GetLatestAttestation(validator)
	if err != nil {
		return nil, err
	}
	bl, err := b.DB.GetBlockForHash(att.Data.BeaconBlockHash)
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
		DB:     db,
		config: config,
		View:   NewBlockchainView(),
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

	err = b.DB.SetBlock(block0)
	if err != nil {
		return nil, err
	}

	// this is a new database, so let's populate it with default values
	node, err := b.View.Index.AddBlockNodeToIndex(&block0, blockHash)
	if err != nil {
		return nil, err
	}

	_, err = b.DB.GetBlockNode(blockHash)
	if err != nil {
		b.View.Chain.SetTip(node)

		b.View.SetFinalizedHead(blockHash, initialState)
		b.View.SetJustifiedHead(blockHash, initialState)
		err = b.DB.SetJustifiedHead(node.Hash)
		if err != nil {
			return nil, err
		}
		err = b.DB.SetJustifiedState(initialState)
		if err != nil {
			return nil, err
		}
		err = b.DB.SetFinalizedHead(node.Hash)
		if err != nil {
			return nil, err
		}
		err = b.DB.SetFinalizedState(initialState)
		if err != nil {
			return nil, err
		}
		b.View.Chain.SetTip(node)

		err = b.stateManager.SetBlockState(blockHash, &initialState)
		if err != nil {
			return nil, err
		}

		err = b.DB.SetHeadBlock(blockHash)
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

			err = b.DB.SetHeadBlock(head.Hash)
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

// GetBlockByHash gets a block by Hash.
func (b *Blockchain) GetBlockByHash(h chainhash.Hash) (*primitives.Block, error) {
	block, err := b.DB.GetBlockForHash(h)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GetConfig returns the config used by this blockchain
func (b *Blockchain) GetConfig() *config.Config {
	return b.config
}

// GetCurrentSlot gets the current slot according to the time.
func (b *Blockchain) GetCurrentSlot() uint64 {
	currentTime := uint64(time.Now().Unix())

	timeSinceGenesis := currentTime - b.stateManager.GetGenesisTime()

	return timeSinceGenesis / uint64(b.config.SlotDuration)
}

// GenesisHash gets the genesis hash for the chain.
func (b *Blockchain) GenesisHash() chainhash.Hash {
	return b.View.Chain.GetBlockByHeight(0).Hash
}
