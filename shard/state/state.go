package state

import (
	"fmt"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
)

// FullShardState is the full state of a shard.
type FullShardState struct {
	tree csmt.Tree
}

// NewFullShardState creates a new empty state tree.
func NewFullShardState(treeDB csmt.TreeDatabase, kvDB csmt.KVStore) *FullShardState {
	return &FullShardState{
		tree: csmt.NewTree(treeDB, kvDB),
	}
}

// Set sets a key in state.
func (s *FullShardState) Set(key chainhash.Hash, value chainhash.Hash) error {
	s.tree.Set(key, value)
	return nil
}

// Get gets a key from the state.
func (s *FullShardState) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	return s.tree.Get(key), nil
}

// SetWithWitness sets a key and generates a witness.
func (s *FullShardState) SetWithWitness(key chainhash.Hash, value chainhash.Hash) *csmt.UpdateWitness {
	return s.tree.SetWithWitness(key, value)
}

// VerifyWitness proves a key in the tree.
func (s *FullShardState) VerifyWitness(key chainhash.Hash) *csmt.VerificationWitness {
	return s.tree.Prove(key)
}

// Hash returns the hash of the tree.
func (s *FullShardState) Hash() chainhash.Hash {
	return s.tree.Hash()
}

// PartialShardState is the partial state of a shard used for executing transactions without the entire state root.
type PartialShardState struct {
	stateRoot             chainhash.Hash
	verificationWitnesses []csmt.VerificationWitness
	updateWitnesses       []csmt.UpdateWitness
}

// Hash gets the hash of the partial state tree.
func (p *PartialShardState) Hash() chainhash.Hash {
	return p.stateRoot
}

// NewPartialShardState creates a new partial shard state.
func NewPartialShardState(root chainhash.Hash, vWitnesses []csmt.VerificationWitness, uWitnesses []csmt.UpdateWitness) *PartialShardState {
	return &PartialShardState{
		stateRoot:             root,
		verificationWitnesses: vWitnesses,
		updateWitnesses:       uWitnesses,
	}
}

// Get gets a key from the partial state.
func (p *PartialShardState) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	if len(p.verificationWitnesses) == 0 {
		return nil, fmt.Errorf("no witnesses left")
	}

	vw := p.verificationWitnesses[0]

	if vw.Key != key {
		return nil, fmt.Errorf("incorrect witness; expected key: %s, got key: %s", vw.Key, key)
	}

	pass := vw.Check(p.stateRoot)
	if !pass {
		return nil, fmt.Errorf("witness did not validate for key: %s", key)
	}

	p.verificationWitnesses = p.verificationWitnesses[1:]

	return &vw.Value, nil
}

// Set sets a key from the partial state.
func (p *PartialShardState) Set(key chainhash.Hash, value chainhash.Hash) error {
	if len(p.updateWitnesses) == 0 {
		return fmt.Errorf("no witnesses left")
	}

	uw := p.updateWitnesses[0]

	if uw.Key != key {
		return fmt.Errorf("incorrect witness; expected key: %s, got key: %s", uw.Key, key)
	}

	if uw.NewValue != value {
		return fmt.Errorf("incorrect witness; expected key: %s, got key: %s", uw.NewValue, value)
	}

	newStateRoot, err := uw.Apply(p.stateRoot)
	if err != nil {
		return err
	}

	p.stateRoot = *newStateRoot
	p.updateWitnesses = p.updateWitnesses[1:]

	return nil
}

// TrackingState keeps track of gets/sets and allows extracting witnesses from some operations.
type TrackingState struct {
	fullState             *FullShardState
	verificationWitnesses []csmt.VerificationWitness
	updateWitnesses       []csmt.UpdateWitness
	initialRoot           chainhash.Hash
}

// Hash returns the hash of the state tree.
func (t *TrackingState) Hash() chainhash.Hash {
	return t.Hash()
}

// NewTrackingState creates a new tracking state derived from a full state.
func NewTrackingState(state *FullShardState) *TrackingState {
	return &TrackingState{
		fullState:   state,
		initialRoot: state.Hash(),
	}
}

// Get gets a key from state.
func (t *TrackingState) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	proof := t.fullState.VerifyWitness(key)
	t.verificationWitnesses = append(t.verificationWitnesses, *proof)
	return t.fullState.Get(key)
}

// Set sets a key in state.
func (t *TrackingState) Set(key chainhash.Hash, value chainhash.Hash) error {
	proof := t.fullState.SetWithWitness(key, value)
	t.updateWitnesses = append(t.updateWitnesses, *proof)
	return nil
}

// GetWitnesses gets the witnesses of the state updates since the last clear.
func (t *TrackingState) GetWitnesses() (chainhash.Hash, []csmt.VerificationWitness, []csmt.UpdateWitness) {
	return t.initialRoot, t.verificationWitnesses, t.updateWitnesses
}

// Reset resets all of the witnesses, but not the state.
func (t *TrackingState) Reset() {
	t.updateWitnesses = nil
	t.verificationWitnesses = nil
}

// AccessInterface is the interface for accessing state through either a full or partial state tree.
type AccessInterface interface {
	Get(key chainhash.Hash) (*chainhash.Hash, error)
	Set(key chainhash.Hash, value chainhash.Hash) error
	Hash() chainhash.Hash
}

var _ AccessInterface = &FullShardState{}
var _ AccessInterface = &TrackingState{}
var _ AccessInterface = &PartialShardState{}
