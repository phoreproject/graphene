package state

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"

	"github.com/phoreproject/synapse/csmt"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/phoreproject/synapse/chainhash"
)

// Shard represents a single shard on the Phore blockchain.
type Shard struct {
	Module           api.Module
	Storage          csmt.TreeTransactionAccess
	ExportedFuncs    []int64
	ExecutionContext ArgumentContext
	Number           uint32

	err           error
	executionLock *sync.Mutex
}

// GetStorageHashForPath gets the hash for a certain path.
func GetStorageHashForPath(path ...[]byte) chainhash.Hash {
	if len(path) == 0 {
		return chainhash.HashH([]byte{})
	} else if len(path) == 1 {
		return chainhash.HashH(path[0])
	} else {
		subHash := GetStorageHashForPath(path[1:]...)
		return chainhash.HashH(append(path[0], subHash[:]...))
	}
}

// ShardNumber gets the current shard number.
func (s *Shard) ShardNumber() int32 {
	return int32(s.Number)
}

// error informs the executor that execution has failed.
func (s *Shard) error(err error) {
	s.err = err
	s.Module.Close(context.TODO())
}

// Load loads a value from state storage.
func (s *Shard) Load(outAddr int32, inAddr int32) {
	addrHash := s.ReadHashAt(inAddr)
	outHash, err := s.Storage.Get(addrHash)
	if err != nil {
		s.error(err)
	}
	s.SafeWrite(outAddr, outHash[:])
}

// Store stores a value to state storage.
func (s *Shard) Store(addr int32, val int32) {
	addrHash := s.ReadHashAt(addr)
	valHash := s.ReadHashAt(val)
	err := s.Storage.Set(addrHash, valHash)
	if err != nil {
		s.error(err)
	}
}

// LoadArgument loads an argument passed in via the transaction.
func (s *Shard) LoadArgument(argNum int32, argLen int32, outAddr int32) {
	out, err := s.ExecutionContext.LoadArgument(argNum, argLen)
	if err != nil {
		s.error(err)
		return
	}

	s.SafeWrite(outAddr, out)
}

// ValidateECDSA validates a certain ECDSA signature.
func (s *Shard) ValidateECDSA(hashAddr int32, signatureAddr int32, pubkeyOut int32) int64 {
	var hash [32]byte
	s.SafeRead(hashAddr, hash[:])

	var sigBytes [65]byte
	s.SafeRead(signatureAddr, sigBytes[:])

	// as long as hash is a random oracle (not chosen by the user),
	// we can derive the pubkey from the signature and the hash.
	pub, _, err := secp256k1.RecoverCompact(sigBytes[:], hash[:])
	if err != nil {
		return 1
	}

	var pubkeyCompressed [33]byte

	copy(pubkeyCompressed[:], pub.Serialize())

	s.SafeWrite(pubkeyOut, pubkeyCompressed[:])

	sig := DecompressSignature(sigBytes)

	if !sig.Verify(hash[:], pub) {
		return 2
	}
	return 0
}

// Hash calculates the 32-byte hash of some data.
func (s *Shard) Hash(hashOut int32, inputStart int32, inputSize int32) {
	toHash := make([]byte, inputSize)

	s.SafeRead(inputStart, toHash)

	h := chainhash.HashH(toHash)

	s.SafeWrite(hashOut, h[:])
}

// Log logs a message.
func (s *Shard) Log(strPtr int32, strLen int32) {
	logMessage := make([]byte, strLen)

	s.SafeRead(strPtr, logMessage)
	fmt.Printf("log message: %s\n", string(logMessage))
}

// DecompressSignature decompresses a signature.
func DecompressSignature(sig [65]byte) *secp256k1.Signature {
	R := new(big.Int)
	S := new(big.Int)

	R.SetBytes(sig[1:33])
	S.SetBytes(sig[33:65])

	return secp256k1.NewSignature(R, S)
}

var _ ShardInterface = &Shard{}

// NewShard creates a new shard given some WASM code, exported funcs, and storage backend.
func NewShard(wasmCode []byte, exportedFuncs []int64, storageAccess csmt.TreeTransactionAccess, shardNumber uint32) (*Shard, error) {
	buf := bytes.NewBuffer(wasmCode)

	s := &Shard{
		Storage:       storageAccess,
		ExportedFuncs: exportedFuncs,
		executionLock: new(sync.Mutex),
		Number:        shardNumber,
	}

	ctx := context.Background()

	runtime := wazero.NewRuntime(ctx)

	_, err := runtime.NewHostModuleBuilder("phore").
		NewFunctionBuilder().WithFunc(s.Load).Export("load").
		NewFunctionBuilder().WithFunc(s.Store).Export("store").
		NewFunctionBuilder().WithFunc(s.ValidateECDSA).Export("validateECDSA").
		NewFunctionBuilder().WithFunc(s.Hash).Export("hash").
		NewFunctionBuilder().WithFunc(s.LoadArgument).Export("loadArgument").
		NewFunctionBuilder().WithFunc(s.Log).Export("write_log").
		NewFunctionBuilder().WithFunc(s.ShardNumber).Export("shard_number").
		Instantiate(ctx, runtime)
	if err != nil {
		return nil, err
	}

	mod, err := runtime.InstantiateModuleFromBinary(ctx, buf.Bytes())
	if err != nil {
		return nil, err
	}

	s.Module = mod

	return s, nil
}

// SafeRead reads from a process and handles errors while reading.
func (s *Shard) SafeRead(location int32, out []byte) {
	bytesRead, ok := s.Module.Memory().Read(uint32(location), uint32(len(out)))
	if !ok {
		s.error(fmt.Errorf("SafeRead read out of bounds (addr: %x, len: %d)", location, len(out)))
	}

	copy(out, bytesRead)
}

// SafeWrite writes to a process and handles errors while writing.
func (s *Shard) SafeWrite(location int32, val []byte) {
	ok := s.Module.Memory().Write(uint32(location), val)
	if !ok {
		s.error(fmt.Errorf("SafeWrite write out of bounds (addr: %x, len: %d)", location, len(val)))
	}
}

// ReadHashAt reads a hash from WebAssembly memory.
func (s *Shard) ReadHashAt(addr int32) [32]byte {
	var hash [32]byte
	s.SafeRead(addr, hash[:])
	return hash
}

// RunFunc runs a shard function
func (s *Shard) RunFunc(executionContext ArgumentContext) error {
	// TODO: ensure in exportedFuncs

	s.executionLock.Lock()
	defer s.executionLock.Unlock()

	fnName := executionContext.GetFunction()

	ctx := context.Background()

	_, err := s.Module.ExportedFunction(fnName).Call(ctx)
	if err != nil {
		return err
	}

	// set this to nil because we should re-set it every time we run a function
	s.ExecutionContext = nil

	if err != nil {
		return err
	}

	if s.err != nil {
		return s.err
	}

	return nil
}

// HashTo64 converts a hash to a uint64.
func HashTo64(h chainhash.Hash) uint64 {
	return binary.BigEndian.Uint64(h[24:])
}

// Uint64ToHash converts a uint64 to a hash.
func Uint64ToHash(i uint64) chainhash.Hash {
	var h chainhash.Hash
	binary.BigEndian.PutUint64(h[24:], i)
	return h
}
