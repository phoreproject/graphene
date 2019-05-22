package beacon

import (
	"errors"
	"time"

	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/utils"

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

// GetEpochBoundaryHash gets the Hash of the parent block at the epoch boundary.
func (b *Blockchain) GetEpochBoundaryHash(slot uint64) (chainhash.Hash, error) {
	epochBoundaryHeight := slot - (slot % b.config.EpochLength)
	bl, err := b.View.Chain.GetBlockBySlot(epochBoundaryHeight)
	if err != nil {
		return chainhash.Hash{}, err
	}
	return bl.Hash, nil
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

// InitialValidatorEntryAndPrivateKey is an initial validator entry and private key.
type InitialValidatorEntryAndPrivateKey struct {
	Entry InitialValidatorEntry

	PrivateKey [32]byte
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
	currentTime := uint64(utils.Now().Unix())

	timeSinceGenesis := currentTime - b.stateManager.GetGenesisTime()

	return timeSinceGenesis / uint64(b.config.SlotDuration)
}

// GenesisHash gets the genesis hash for the chain.
func (b *Blockchain) GenesisHash() chainhash.Hash {
	return b.View.Chain.GetBlockByHeight(0).Hash
}
