package beacon

import (
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/primitives/proofs"
	"github.com/phoreproject/synapse/utils"
	"github.com/prysmaticlabs/go-ssz"

	"github.com/phoreproject/synapse/chainhash"
)

var zeroHash = chainhash.Hash{}

// Blockchain handles the communication between 3 main components.
// 1. BlockchainView stores block nodes as a chain and an index to allow
//    easy access. All block nodes are stored in memory and only store
//    information needed often.
// 2. DB keeps track of the persistent data. It stores large block
//    files, a representation of the chain through block nodes, and
//    important information about the state of the chain.
// 3. StateManager stores information about the state of certain blocks.
//    The state manager stores the state and derived states of every block
//    after the finalized block.
type Blockchain struct {
	View         *BlockchainView
	DB           db.Database
	config       *config.Config
	stateManager *StateManager

	notifees    []BlockchainNotifee
	notifeeLock sync.Mutex
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
	h, _ := ssz.HashTreeRoot(bl)

	node := b.View.Index.GetBlockNodeByHash(h)
	if node == nil {
		return nil, errors.New("couldn't find block attested to by validator in index")
	}
	return node, nil
}

// NewBlockchainWithInitialValidators creates a new blockchain with the specified
// initial validators.
func NewBlockchainWithInitialValidators(db db.Database, config *config.Config, validators []primitives.InitialValidatorEntry, skipValidation bool, genesisTime uint64) (*Blockchain, error) {
	b := &Blockchain{
		DB:     db,
		config: config,
		View:   NewBlockchainView(),
	}

	sm, err := NewStateManager(config, genesisTime, b, db)
	if err != nil {
		return nil, err
	}

	b.stateManager = sm

	initialState, err := primitives.InitializeState(config, validators, genesisTime, skipValidation)
	if err != nil {
		return nil, err
	}

	stateRoot, err := ssz.HashTreeRoot(initialState)
	if err != nil {
		return nil, err
	}

	block0 := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:     0,
			StateRoot:      stateRoot,
			ParentRoot:     zeroHash,
			RandaoReveal:   bls.EmptySignature.Serialize(),
			Signature:      bls.EmptySignature.Serialize(),
			ValidatorIndex: 0,
		},
		BlockBody: primitives.BlockBody{
			ProposerSlashings: []primitives.ProposerSlashing{},
			CasperSlashings:   []primitives.CasperSlashing{},
			Attestations:      []primitives.Attestation{},
			Deposits:          []primitives.Deposit{},
			Exits:             []primitives.Exit{},
		},
	}

	blockHash, err := ssz.HashTreeRoot(block0)
	if err != nil {
		return nil, err
	}

	logrus.WithField("genesisHash", chainhash.Hash(blockHash)).Info("initializing blockchain with genesis block")

	err = b.DB.SetBlock(block0)
	if err != nil {
		return nil, err
	}

	// this is a new database, so let's populate it with default values
	node, err := b.View.Index.AddBlockNodeToIndex(&block0, blockHash, stateRoot, proofs.GetValidatorHash(initialState, config.EpochLength))
	if err != nil {
		return nil, err
	}

	// check if the block index exists in the database
	_, err = b.DB.GetBlockNode(blockHash)
	if err != nil {
		// if it doesn't, initialize the database
		if err := b.initializeDatabase(node, *initialState); err != nil {
			return nil, err
		}
	} else {
		// if it does, load everything needed from disk
		if err := b.loadBlockchainFromDisk(blockHash); err != nil {
			return nil, err
		}
	}

	return b, nil
}

// GetUpdatedState gets the tip, but with processed slots/epochs as appropriate.
func (b *Blockchain) GetUpdatedState(upTo uint64) (*primitives.State, error) {
	tip := b.View.Chain.Tip()

	view := NewChainView(tip)

	_, tipState, err := b.stateManager.GetStateForHashAtSlot(tip.Hash, upTo, &view, b.config)
	if err != nil {
		return nil, err
	}

	tipStateCopy := tipState.Copy()

	return &tipStateCopy, nil
}

// GetNextSlotTime returns the timestamp of the next slot.
func (b *Blockchain) GetNextSlotTime() time.Time {
	return time.Unix(int64((b.View.Chain.Tip().Slot+1)*uint64(b.config.SlotDuration)+b.stateManager.GetGenesisTime()), 0)
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

// GetGenesisTime gets the genesis time for this blockchain.
func (b *Blockchain) GetGenesisTime() uint64 {
	return b.stateManager.GetGenesisTime()
}
