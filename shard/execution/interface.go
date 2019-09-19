package execution

import (
	"fmt"
	"github.com/go-interpreter/wagon/exec"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/shard/state"
)

// ShardInterface is the interface a shard uses to access external information like state, and access slow functions like
// validating signatures and hashing data.
type ShardInterface interface {
	Load(proc *exec.Process, outAddr int32, inAddr int32)
	Store(proc *exec.Process, addr int32, val int32)
	LoadArgument(proc *exec.Process, argNum int32, argLen int32, outAddr int32)
	ValidateECDSA(proc *exec.Process, hashAddr int32, signatureAddr int32, pubkeyOut int32) int64
	Hash(proc *exec.Process, hashOut int32, inputStart int32, inputSize int32)
	Log(proc *exec.Process, strPtr int32, len int32)
}

// TransitionInterface executes some transactions using either a transaction package or the entire state.
type TransitionInterface interface {
	// Transition calculates the transition from one state root to the next.
	Transition(chainhash.Hash) (*chainhash.Hash, error)
}

// FullStateTransition transitions the state using the transaction provided.
type FullStateTransition struct {
	state        *state.FullShardState
	transactions [][]byte
	code         []byte
	shardID      uint32
}

// NewFullStateTransition creates a new full state transition using the previous state and the transactions to run.
func NewFullStateTransition(state *state.FullShardState, transactions [][]byte, code []byte, shardID uint32) *FullStateTransition {
	return &FullStateTransition{
		state:        state,
		transactions: transactions,
		code:         code,
		shardID:      shardID,
	}
}

// Transition runs the transition using the given prehash to calculate the post hash.
func (f *FullStateTransition) Transition(preHash *chainhash.Hash) (*chainhash.Hash, error) {
	expectedPreHash := f.state.Hash()
	if !preHash.IsEqual(&expectedPreHash) {
		return nil, fmt.Errorf("expected state root of full state to equal %s but got %s", expectedPreHash, preHash)
	}

	shard, err := NewShard(f.code, []int64{}, f.state, f.shardID)
	if err != nil {
		return nil, err
	}

	for _, tx := range f.transactions {
		argContext, err := LoadArgumentContextFromTransaction(tx)
		if err != nil {
			return nil, err
		}

		out, err := shard.RunFunc(argContext)
		if err != nil {
			return nil, err
		}

		if out.(uint64) != 0 {
			return nil, fmt.Errorf("transaction failed with code: %d", out)
		}
	}

	postHash := f.state.Hash()

	return &postHash, nil
}

// GetPostState gets the state after the state transition.
func (f *FullStateTransition) GetPostState() *state.FullShardState {
	return f.state
}
