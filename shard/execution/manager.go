package execution

import (
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/shard/state"
)

// BasicFullStateManager is a manager for shard's full state with no support for reorgs or roll-backs. It assumes that the
// next call to Transition will use the same prehash as the last posthash.
type BasicFullStateManager struct {
	state   *state.FullShardState
	shardID uint32
	code    []byte
}

// NewBasicFullStateManager creates a new basic full state manager with the given code.
func NewBasicFullStateManager(code []byte, shardID uint32) *BasicFullStateManager {
	return &BasicFullStateManager{
		state:   state.NewFullShardState(),
		code:    code,
		shardID: shardID,
	}
}

// Transition transitions from one hash to the next given a list of transactions.
func (m *BasicFullStateManager) Transition(preHash chainhash.Hash, transactions [][]byte) (*chainhash.Hash, error) {
	fst := NewFullStateTransition(m.state, transactions, m.code, m.shardID)
	postHash, err := fst.Transition(&preHash)
	if err != nil {
		return nil, err
	}
	m.state = fst.GetPostState()
	return postHash, nil
}

// CheckTransition gets the state root without modifying the current state.
func (m *BasicFullStateManager) CheckTransition(preHash chainhash.Hash, transactions [][]byte) (*chainhash.Hash, error) {
	stateCopy := m.state.Copy()
	fst := NewFullStateTransition(stateCopy, transactions, m.code, m.shardID)
	postHash, err := fst.Transition(&preHash)
	if err != nil {
		return nil, err
	}
	return postHash, nil
}

// Get gets a key from the current state.
func (m *BasicFullStateManager) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	return m.state.Get(key)
}
