package beacon

import (
	"fmt"
	"sync"

	"github.com/phoreproject/synapse/beacon/db"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

type stateDerivedFromBlock struct {
	slotsAfterBlock []*primitives.State // slotsAfterBlock[n].Slot = blockSlot + n
	firstSlot       uint64
	lock            *sync.Mutex
}

func newStateDerivedFromBlock(stateAfterProcessingBlock *primitives.State) *stateDerivedFromBlock {
	return &stateDerivedFromBlock{
		slotsAfterBlock: []*primitives.State{stateAfterProcessingBlock},
		firstSlot:       stateAfterProcessingBlock.Slot,
		lock:            new(sync.Mutex),
	}
}

func (s *stateDerivedFromBlock) deriveState(slot uint64, view primitives.BlockView, c *config.Config) (*primitives.State, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	slotIndex := slot - s.firstSlot

	if slotIndex < uint64(len(s.slotsAfterBlock)) {
		return s.slotsAfterBlock[slotIndex], nil
	}
	for generatingSlot := len(s.slotsAfterBlock); uint64(generatingSlot) <= slotIndex; generatingSlot++ {
		bestState := s.slotsAfterBlock[len(s.slotsAfterBlock)-1].Copy()

		err := bestState.ProcessSlots(s.firstSlot+uint64(generatingSlot), view, c)
		if err != nil {
			return nil, err
		}

		s.slotsAfterBlock = append(s.slotsAfterBlock, &bestState)
	}

	return s.slotsAfterBlock[slotIndex], nil
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
	return derivedState.slotsAfterBlock[0], true
}

// GetStateForHashAtSlot gets the state derived from a certain block Hash at a given
// slot.
func (sm *StateManager) GetStateForHashAtSlot(blockHash chainhash.Hash, slot uint64, view primitives.BlockView, c *config.Config) (*primitives.State, error) {
	sm.stateMapLock.RLock()
	derivedState, found := sm.stateMap[blockHash]
	sm.stateMapLock.RUnlock()
	if !found {
		return nil, fmt.Errorf("could not find state for block %s", blockHash)
	}

	return derivedState.deriveState(slot, view, c)
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

	lastBlockState, err := sm.GetStateForHashAtSlot(lastBlockHash, block.BlockHeader.SlotNumber, &view, sm.config)
	if err != nil {
		return nil, nil, err
	}

	newState := lastBlockState.Copy()

	err = newState.ProcessBlock(block, sm.config, &view, verifySignature)
	if err != nil {
		return nil, nil, err
	}

	var receipts []primitives.Receipt

	if newState.Slot/sm.config.EpochLength > newState.EpochIndex && newState.Slot%sm.config.EpochLength == 0 {
		receipts, err = newState.ProcessEpochTransition(sm.config, &view)
		if err != nil {
			return nil, nil, err
		}
	}

	blockHash, err := ssz.TreeHash(block)
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
