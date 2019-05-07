package beacon

import (
	"errors"
	"fmt"
	"github.com/phoreproject/synapse/beacon/db"
	"sync"
	"time"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// StateManager handles all state transitions, storing of states for different forks,
// and time-based state updates.
type StateManager struct {
	config       *config.Config
	state        primitives.State // state of the head
	stateMap     map[chainhash.Hash]primitives.State
	db           db.Database
	blockchain   *Blockchain
	stateMapLock *sync.RWMutex
	stateLock    *sync.Mutex
	genesisTime  uint64
}

// NewStateManager creates a new state manager.
func NewStateManager(c *config.Config, initialValidators []InitialValidatorEntry, genesisTime uint64, skipValidation bool, blockchain *Blockchain, db db.Database) (*StateManager, error) {
	s := &StateManager{
		config:       c,
		stateMap:     make(map[chainhash.Hash]primitives.State),
		stateMapLock: new(sync.RWMutex),
		stateLock:    new(sync.Mutex),
		genesisTime:  genesisTime,
		blockchain:   blockchain,
		db:           db,
	}
	intialState, err := InitializeState(c, initialValidators, genesisTime, skipValidation)
	if err != nil {
		return nil, err
	}
	s.state = *intialState
	return s, nil
}

// GetGenesisTime gets the time of the genesis slot.
func (sm *StateManager) GetGenesisTime() uint64 {
	return sm.genesisTime
}

// GetStateForHash gets the state for a certain block Hash.
func (sm *StateManager) GetStateForHash(blockHash chainhash.Hash) (*primitives.State, bool) {
	sm.stateMapLock.RLock()
	state, found := sm.stateMap[blockHash]
	sm.stateMapLock.RUnlock()
	if !found {
		return nil, false
	}
	return &state, true
}

// UpdateHead updates the head state to a state in the stateMap.
func (sm *StateManager) UpdateHead(blockHash chainhash.Hash) error {
	sm.stateMapLock.RLock()
	state, found := sm.stateMap[blockHash]
	if !found {
		return fmt.Errorf("couldn't find block with Hash %s in state map", blockHash)
	}
	sm.stateMapLock.RUnlock()
	sm.stateLock.Lock()
	logrus.WithField("slot", state.Slot).Debug("setting state head")
	sm.state = state
	sm.stateLock.Unlock()
	return nil
}

// GetHeadSlot gets the slot of the head state.
func (sm *StateManager) GetHeadSlot() uint64 {
	return sm.state.Slot
}

// GetHeadState gets the head state.
func (sm *StateManager) GetHeadState() primitives.State {
	return sm.state
}

// InitializeState initializes state to the genesis state according to the config.
func InitializeState(c *config.Config, initialValidators []InitialValidatorEntry, genesisTime uint64, skipValidation bool) (*primitives.State, error) {
	crosslinks := make([]primitives.Crosslink, c.ShardCount)

	for i := 0; i < c.ShardCount; i++ {
		crosslinks[i] = primitives.Crosslink{
			Slot:           c.InitialSlotNumber,
			ShardBlockHash: zeroHash,
		}
	}

	recentBlockHashes := make([]chainhash.Hash, c.LatestBlockRootsLength)
	for i := uint64(0); i < c.LatestBlockRootsLength; i++ {
		recentBlockHashes[i] = zeroHash
	}

	initialState := primitives.State{
		Slot:        0,
		EpochIndex:  0,
		GenesisTime: genesisTime,
		ForkData: primitives.ForkData{
			PreForkVersion:  c.InitialForkVersion,
			PostForkVersion: c.InitialForkVersion,
			ForkSlotNumber:  c.InitialSlotNumber,
		},
		ValidatorRegistry:                 []primitives.Validator{},
		ValidatorBalances:                 []uint64{},
		ValidatorRegistryLatestChangeSlot: c.InitialSlotNumber,
		ValidatorRegistryExitCount:        0,
		ValidatorRegistryDeltaChainTip:    chainhash.Hash{},

		RandaoMix:                 chainhash.Hash{},
		NextSeed:                  chainhash.Hash{},
		ShardAndCommitteeForSlots: [][]primitives.ShardAndCommittee{},

		PreviousJustifiedSlot: c.InitialSlotNumber,
		JustifiedSlot:         c.InitialSlotNumber,
		JustificationBitfield: 0,
		FinalizedSlot:         c.InitialSlotNumber,

		LatestCrosslinks:            crosslinks,
		LatestBlockHashes:           recentBlockHashes,
		LatestPenalizedExitBalances: []uint64{},
		LatestAttestations:          []primitives.PendingAttestation{},
		BatchedBlockRoots:           []chainhash.Hash{},
	}

	for _, deposit := range initialValidators {
		pub, err := bls.DeserializePublicKey(deposit.PubKey)
		if err != nil {
			return nil, err
		}
		validatorIndex, err := initialState.ProcessDeposit(pub, deposit.DepositSize, deposit.ProofOfPossession, deposit.WithdrawalCredentials, skipValidation, c)
		if err != nil {
			return nil, err
		}
		if initialState.GetEffectiveBalance(validatorIndex, c) == c.MaxDeposit {
			err := initialState.UpdateValidatorStatus(validatorIndex, primitives.Active, c)
			if err != nil {
				return nil, err
			}
		}
	}

	initialShuffling := primitives.GetNewShuffling(zeroHash, initialState.ValidatorRegistry, 0, c)
	initialState.ShardAndCommitteeForSlots = append(initialShuffling, initialShuffling...)

	return &initialState, nil
}

// SetBlockState sets the state for a certain block. This SHOULD ONLY
// BE USED FOR THE GENESIS BLOCK!
func (sm *StateManager) SetBlockState(blockHash chainhash.Hash, state *primitives.State) error {
	// add the state to the statemap
	sm.stateMapLock.Lock()
	defer sm.stateMapLock.Unlock()
	logrus.WithField("hash", blockHash.String()).Debug("setting block state")
	sm.stateMap[blockHash] = *state
	return nil
}

// AddBlockToStateMap processes the block and adds it to the state map.
func (sm *StateManager) AddBlockToStateMap(block *primitives.Block, verifySignature bool) (*primitives.State, error) {
	lastBlockHash := block.BlockHeader.ParentRoot

	view, err := sm.blockchain.GetSubView(lastBlockHash)
	if err != nil {
		return nil, err
	}

	lastBlockState, found := sm.GetStateForHash(lastBlockHash)
	if !found {
		return nil, errors.New("could not find block state of parent block")
	}

	newState := lastBlockState.Copy()

	err = newState.ProcessSlots(block.BlockHeader.SlotNumber, &view, sm.config)
	if err != nil {
		return nil, err
	}

	err = newState.ProcessBlock(block, sm.config, &view, verifySignature)
	if err != nil {
		return nil, err
	}

	if newState.Slot/sm.config.EpochLength > newState.EpochIndex && newState.Slot%sm.config.EpochLength == 0 {
		logrus.Info("processing epoch transition")
		t := time.Now()

		err := newState.ProcessEpochTransition(sm.config, &view)
		if err != nil {
			return nil, err
		}
		logrus.WithField("time", time.Since(t)).Debug("done processing epoch transition")
	}

	blockHash, err := ssz.TreeHash(block)
	if err != nil {
		return nil, err
	}

	err = sm.SetBlockState(blockHash, &newState)
	if err != nil {
		return nil, err
	}

	return &newState, nil
}

// DeleteStateBeforeFinalizedSlot deletes any states before the current finalized slot.
func (sm *StateManager) DeleteStateBeforeFinalizedSlot(finalizedSlot uint64) error {
	sm.stateMapLock.Lock()
	defer sm.stateMapLock.Unlock()
	// once we finalize the block, we can get rid of any states before finalizedSlot
	for i := range sm.stateMap {
		// if it happened before finalized slot, we don't need it
		if sm.stateMap[i].Slot < finalizedSlot {
			delete(sm.stateMap, i)
		}
	}
	return nil
}
