package beacon

import (
	"fmt"
	"sync"

	"github.com/phoreproject/synapse/beacon/db"

	"github.com/phoreproject/synapse/beacon/config"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/go-ssz"
	logger "github.com/sirupsen/logrus"
)

type stateDerivedFromBlock struct {
	firstSlot      uint64
	firstSlotState *primitives.State

	lastSlot         uint64
	lastSlotState    *primitives.State
	lastSlotReceipts []primitives.Receipt

	lock *sync.Mutex
}

func newStateDerivedFromBlock(stateAfterProcessingBlock *primitives.State) *stateDerivedFromBlock {
	return &stateDerivedFromBlock{
		firstSlotState: stateAfterProcessingBlock,
		firstSlot:      stateAfterProcessingBlock.Slot,
		lastSlotState:  stateAfterProcessingBlock,
		lastSlot:       stateAfterProcessingBlock.Slot,
		lock:           new(sync.Mutex),
	}
}

func (s *stateDerivedFromBlock) deriveState(slot uint64, view primitives.BlockView, c *config.Config) ([]primitives.Receipt, *primitives.State, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if slot == s.lastSlot {
		return s.lastSlotReceipts, s.lastSlotState, nil
	}

	if slot < s.lastSlot {
		derivedState := s.firstSlotState.Copy()

		receipts, err := derivedState.ProcessSlots(slot, view, c)
		if err != nil {
			return nil, nil, err
		}

		view.SetTipSlot(slot)

		return receipts, &derivedState, nil
	}

	view.SetTipSlot(s.lastSlot)

	receipts, err := s.lastSlotState.ProcessSlots(slot, view, c)
	if err != nil {
		return nil, nil, err
	}

	s.lastSlot = slot
	s.lastSlotReceipts = append(s.lastSlotReceipts, receipts...)

	return s.lastSlotReceipts, s.lastSlotState, nil
}

// StateManager handles all state transitions, storing of states for different forks,
// and time-based state updates.
type StateManager struct {
	config *config.Config

	// stateMap maps block hashes to states where the last block processed is that block
	// hash. The internal store tracks slots processed
	stateMap     map[chainhash.Hash]*stateDerivedFromBlock
	db           db.Database
	blockchain   *Blockchain
	stateMapLock *sync.RWMutex
	stateLock    *sync.Mutex
	genesisTime  uint64
}

// NewStateManager creates a new state manager.
func NewStateManager(c *config.Config, genesisTime uint64, blockchain *Blockchain, db db.Database) (*StateManager, error) {
	s := &StateManager{
		config:       c,
		stateMap:     make(map[chainhash.Hash]*stateDerivedFromBlock),
		stateMapLock: new(sync.RWMutex),
		stateLock:    new(sync.Mutex),
		genesisTime:  genesisTime,
		blockchain:   blockchain,
		db:           db,
	}
	return s, nil
}

// GetGenesisTime gets the time of the genesis slot.
func (sm *StateManager) GetGenesisTime() uint64 {
	return sm.genesisTime
}

// GetStateForHash gets the state for a certain block Hash.
func (sm *StateManager) GetStateForHash(blockHash chainhash.Hash) (*primitives.State, bool) {
	sm.stateMapLock.RLock()
	derivedState, found := sm.stateMap[blockHash]
	sm.stateMapLock.RUnlock()
	if !found {
		return nil, false
	}
	derivedState.lock.Lock()
	defer derivedState.lock.Unlock()
	return derivedState.firstSlotState, true
}

// GetStateForHashAtSlot gets the state derived from a certain block Hash at a given
// slot.
func (sm *StateManager) GetStateForHashAtSlot(blockHash chainhash.Hash, slot uint64, view primitives.BlockView, c *config.Config) ([]primitives.Receipt, *primitives.State, error) {
	sm.stateMapLock.RLock()
	derivedState, found := sm.stateMap[blockHash]
	sm.stateMapLock.RUnlock()
	if !found {
		return nil, nil, fmt.Errorf("could not find state for block %s", blockHash)
	}

	receipts, state, err := derivedState.deriveState(slot, view, c)

	logger.Debugf("derivedState.lastSlotReceipts length: %d for block: %s\n", len(derivedState.lastSlotReceipts), blockHash.String())

	return receipts, state, err
}

// SetBlockState sets the state for a certain block. This SHOULD ONLY
// BE USED FOR THE GENESIS BLOCK!
func (sm *StateManager) SetBlockState(blockHash chainhash.Hash, state *primitives.State) error {
	// add the state to the state map
	sm.stateMapLock.Lock()
	defer sm.stateMapLock.Unlock()
	//logrus.WithField("hash", blockHash.String()).Debug("setting block state")
	sm.stateMap[blockHash] = newStateDerivedFromBlock(state)
	return nil
}

// AddBlockToStateMap processes the block and adds it to the state map.
func (sm *StateManager) AddBlockToStateMap(block *primitives.Block, verifySignature bool) ([]primitives.Receipt, *primitives.State, error) {
	lastBlockHash := block.BlockHeader.ParentRoot

	view, err := sm.blockchain.GetSubView(lastBlockHash)
	if err != nil {
		return nil, nil, err
	}

	receipts, lastBlockState, err := sm.GetStateForHashAtSlot(lastBlockHash, block.BlockHeader.SlotNumber, &view, sm.config)
	if err != nil {
		return nil, nil, err
	}

	newState := lastBlockState.Copy()

	err = newState.ProcessBlock(block, sm.config, &view, verifySignature)
	if err != nil {
		return nil, nil, err
	}

	blockHash, err := ssz.HashTreeRoot(block)
	if err != nil {
		return nil, nil, err
	}

	err = sm.SetBlockState(blockHash, &newState)
	if err != nil {
		return nil, nil, err
	}

	return receipts, &newState, nil
}

// DeleteStateBeforeFinalizedSlot deletes any states before the current finalized slot.
func (sm *StateManager) DeleteStateBeforeFinalizedSlot(finalizedSlot uint64) error {
	sm.stateMapLock.Lock()
	defer sm.stateMapLock.Unlock()
	// once we finalize the block, we can get rid of any states before finalizedSlot
	for i := range sm.stateMap {
		// if it happened before finalized slot, we don't need it
		if sm.stateMap[i].firstSlot < finalizedSlot {
			delete(sm.stateMap, i)
		}
	}
	return nil
}
