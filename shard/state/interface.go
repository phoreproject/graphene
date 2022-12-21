package state

import (
	"fmt"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
)

// ShardInterface is the interface a shard uses to access external information like state, and access slow functions like
// validating signatures and hashing data.
type ShardInterface interface {
	Load(outAddr int32, inAddr int32)
	Store(addr int32, val int32)
	LoadArgument(argNum int32, argLen int32, outAddr int32)
	ValidateECDSA(hashAddr int32, signatureAddr int32, pubkeyOut int32) int64
	Hash(hashOut int32, inputStart int32, inputSize int32)
	Log(strPtr int32, len int32)
}

// TransitionInterface executes some transactions using either a transaction package or the entire state.
type TransitionInterface interface {
	// Transition calculates the transition from one state root to the next.
	Transition(chainhash.Hash) (*chainhash.Hash, error)
}

// ShardInfo is all of the shard specific information needed for execution.
type ShardInfo struct {
	CurrentCode []byte
	ShardID     uint32
}

// Transition runs a transaction.
func Transition(state csmt.TreeTransactionAccess, tx []byte, info ShardInfo) (*chainhash.Hash, error) {
	shard, err := NewShard(info.CurrentCode, []int64{}, state, info.ShardID)
	if err != nil {
		return nil, err
	}

	argContext, err := LoadArgumentContextFromTransaction(tx)
	if err != nil {
		return nil, err
	}

	outCode, err := shard.RunFunc(argContext)
	if err != nil {
		return nil, err
	}

	if outCode != 0 {
		return nil, fmt.Errorf("function returned non-zero exit code")
	}

	outHash, err := state.Hash()
	if err != nil {
		return nil, err
	}

	return outHash, nil
}
