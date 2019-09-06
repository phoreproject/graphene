package execution

import "github.com/go-interpreter/wagon/exec"

// ShardInterface is the interface a shard uses to access external information like state, and access slow functions like
// validating signatures and hashing data.
type ShardInterface interface {
	Load(proc *exec.Process, outAddr int32, inAddr int32)
	Store(proc *exec.Process, addr int32, val int32)
	LoadArgument(proc *exec.Process, argNum int32, outAddr int32)
	ValidateECDSA(proc *exec.Process, hashAddr int32, signatureAddr int32, pubkeyOut int32) int64
	Hash(proc *exec.Process, hashOut int32, inputStart int32, inputSize int32)
	Log(proc *exec.Process, strPtr int32, len int32)
}
