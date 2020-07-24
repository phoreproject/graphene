package state

import (
	"fmt"
	"sync"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/primitives"
)

// State is shard state without the database.
type State struct {
	SmartContractRoot chainhash.Hash
	LastCrosslinkHash chainhash.Hash
	ProposerAssignments []bls.PublicKey
}

// FullContractState represents a full shard state.
type FullContractState struct {
	tree csmt.Tree
}

// Hash gets the hash of the current state root.
func (s *FullContractState) Hash() (*chainhash.Hash, error) {
	var out *chainhash.Hash
	err := s.View(func(a AccessInterface) error {
		outHash, err := a.Hash()
		if err != nil {
			return err
		}
		out = outHash

		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, err
}

// NewFullShardState creates a new empty state tree.
func NewFullShardState(tree csmt.Tree) *FullContractState {
	return &FullContractState{tree}
}

// Update updates the underlying state.
func (s *FullContractState) Update(cb func(AccessInterface) error) error {
	return s.tree.Update(func(tx csmt.TreeTransactionAccess) error {
		return cb(newFullShardStateAccess(tx.(*csmt.TreeTransaction)))
	})
}

// View views the underlying state.
func (s *FullContractState) View(cb func(AccessInterface) error) error {
	return s.tree.View(func(tx csmt.TreeTransactionAccess) error {
		return cb(newFullShardStateAccess(tx.(*csmt.TreeTransaction)))
	})
}

// FullShardStateAccess is access to a standard interface to state.
type FullShardStateAccess struct {
	tree *csmt.TreeTransaction
}

func newFullShardStateAccess(treeTX *csmt.TreeTransaction) *FullShardStateAccess {
	return &FullShardStateAccess{
		tree: treeTX,
	}
}

// Set sets a key in state.
func (s *FullShardStateAccess) Set(key chainhash.Hash, value chainhash.Hash) error {
	return s.tree.Set(key, value)
}

// Get gets a key from the state.
func (s *FullShardStateAccess) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	return s.tree.Get(key)
}

// SetWithWitness sets a key and generates a witness.
func (s *FullShardStateAccess) SetWithWitness(key chainhash.Hash, value chainhash.Hash) (*primitives.UpdateWitness, error) {
	return s.tree.SetWithWitness(key, value)
}

// VerifyWitness proves a key in the tree.
func (s *FullShardStateAccess) VerifyWitness(key chainhash.Hash) (*primitives.VerificationWitness, error) {
	return s.tree.Prove(key)
}

// Hash returns the hash of the tree.
func (s *FullShardStateAccess) Hash() (*chainhash.Hash, error) {
	return s.tree.Hash()
}

// PartialShardStateAccess is the partial state of a shard used for executing transactions without the entire state root.
type PartialShardStateAccess struct {
	stateRoot             chainhash.Hash
	verificationWitnesses []primitives.VerificationWitness
	updateWitnesses       []primitives.UpdateWitness
}

// Hash gets the hash of the partial state tree.
func (p PartialShardStateAccess) Hash() (*chainhash.Hash, error) {
	return &p.stateRoot, nil
}

// NewPartialShardState creates a new partial shard state. This isn't transactional because it can't be. All operations need
// to be executed in order.
func NewPartialShardState(root chainhash.Hash, vWitnesses []primitives.VerificationWitness, uWitnesses []primitives.UpdateWitness) *PartialShardStateAccess {
	return &PartialShardStateAccess{
		stateRoot:             root,
		verificationWitnesses: vWitnesses,
		updateWitnesses:       uWitnesses,
	}
}

// Get gets a key from the partial state.
func (p *PartialShardStateAccess) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	if len(p.verificationWitnesses) == 0 {
		return nil, fmt.Errorf("no witnesses left")
	}

	vw := p.verificationWitnesses[0]

	if vw.Key != key {
		return nil, fmt.Errorf("incorrect witness; expected key: %s, got key: %s", vw.Key, key)
	}

	pass := csmt.CheckWitness(&vw, p.stateRoot)
	if !pass {
		return nil, fmt.Errorf("witness did not validate for key: %s", key)
	}

	p.verificationWitnesses = p.verificationWitnesses[1:]

	return &vw.Value, nil
}

// Set sets a key from the partial state.
func (p *PartialShardStateAccess) Set(key chainhash.Hash, value chainhash.Hash) error {
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

	newStateRoot, err := csmt.ApplyWitness(uw, p.stateRoot)
	if err != nil {
		return err
	}

	p.stateRoot = *newStateRoot
	p.updateWitnesses = p.updateWitnesses[1:]

	return nil
}

// TrackingState keeps track of gets/sets and allows extracting witnesses from some operations.
type TrackingState struct {
	fullState             csmt.Tree
	verificationWitnesses []primitives.VerificationWitness
	updateWitnesses       []primitives.UpdateWitness
	initialRoot           chainhash.Hash

	lock *sync.RWMutex
}

// Hash returns the current tree root.
func (t *TrackingState) Hash() (*chainhash.Hash, error) {
	var out *chainhash.Hash
	err := t.View(func(a csmt.TreeTransactionAccess) error {
		outHash, err := a.Hash()
		if err != nil {
			return err
		}
		out = outHash

		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, err
}

// NewTrackingState creates a new tracking state derived from a full state.
func NewTrackingState(state csmt.Tree) (*TrackingState, error) {
	initialRoot, err := state.Hash()
	if err != nil {
		return nil, err
	}

	return &TrackingState{
		fullState:   state,
		initialRoot: initialRoot,
		lock: new(sync.RWMutex),
	}, nil
}

// GetWitnesses gets the witnesses of the state updates since the last clear.
func (t *TrackingState) GetWitnesses() (chainhash.Hash, []primitives.VerificationWitness, []primitives.UpdateWitness) {
	return t.initialRoot, t.verificationWitnesses, t.updateWitnesses
}

// Reset resets all of the witnesses, but not the state.
func (t *TrackingState) Reset() error {
	t.updateWitnesses = nil
	t.verificationWitnesses = nil
	var r *chainhash.Hash
	err := t.fullState.View(func(a csmt.TreeTransactionAccess) error {
		rootHash, err := a.Hash()
		if err != nil {
			return err
		}

		r = rootHash

		return nil
	})
	if err != nil {
		return err
	}
	t.initialRoot = *r

	return nil
}

// Update updates the underlying state and tracks changes.
func (t *TrackingState) Update(cb func(transaction csmt.TreeTransactionAccess) error) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	tsa := &TrackingStateAccess{
		verificationWitnesses: t.verificationWitnesses[:],
		updateWitnesses:       t.updateWitnesses[:],
	}

	err := t.fullState.Update(func(a csmt.TreeTransactionAccess) error {
		nonTracking := newFullShardStateAccess(a.(*csmt.TreeTransaction))
		tsa.fullState = *nonTracking
		return cb(tsa)
	})
	if err != nil {
		return err
	}

	t.updateWitnesses = tsa.updateWitnesses
	t.verificationWitnesses = tsa.verificationWitnesses

	return nil
}

// View gives a consistent view of the underlying state and tracks changes.
func (t *TrackingState) View(cb func(transaction csmt.TreeTransactionAccess) error) error {
	t.lock.RLock()

	tsa := &TrackingStateAccess{
		verificationWitnesses: t.verificationWitnesses[:],
		updateWitnesses:       t.updateWitnesses[:],
	}

	t.lock.RUnlock()

	err := t.fullState.Update(func(a csmt.TreeTransactionAccess) error {
		nonTracking := newFullShardStateAccess(a.(*csmt.TreeTransaction))
		tsa.fullState = *nonTracking
		return cb(tsa)
	})
	if err != nil {
		return err
	}

	return nil
}

// TrackingStateAccess provides access to tracking state.
type TrackingStateAccess struct {
	fullState FullShardStateAccess
	verificationWitnesses []primitives.VerificationWitness
	updateWitnesses []primitives.UpdateWitness
}

// Hash returns the hash of the state tree.
func (t *TrackingStateAccess) Hash() (*chainhash.Hash, error) {
	return t.fullState.Hash()
}


// Get gets a key from state.
func (t *TrackingStateAccess) Get(key chainhash.Hash) (*chainhash.Hash, error) {
	proof, err := t.fullState.VerifyWitness(key)
	if err != nil {
		return nil, err
	}
	t.verificationWitnesses = append(t.verificationWitnesses, *proof)
	return t.fullState.Get(key)
}

// Set sets a key in state.
func (t *TrackingStateAccess) Set(key chainhash.Hash, value chainhash.Hash) error {
	// we know we're getting a full state here so we can do this without checking
	proof, err := t.fullState.SetWithWitness(key, value)
	if err != nil {
		return err
	}
	t.updateWitnesses = append(t.updateWitnesses, *proof)
	return nil
}

// TransactionInterface is an interface that provides transactions to update state.
type TransactionInterface interface {
	Update(func(AccessInterface) error) error
	View(func(AccessInterface) error) error
	Hash() (*chainhash.Hash, error)
}

// AccessInterface is the interface for accessing state through either a full or partial state tree.
type AccessInterface interface {
	Get(key chainhash.Hash) (*chainhash.Hash, error)
	Set(key chainhash.Hash, value chainhash.Hash) error
	Hash() (*chainhash.Hash, error)
}

var _ TransactionInterface = &FullContractState{}
var _ csmt.TreeTransactionAccess = &TrackingStateAccess{}
var _ AccessInterface = &PartialShardStateAccess{}