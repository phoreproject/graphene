package execution

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/phoreproject/synapse/shard/state"
	"github.com/sirupsen/logrus"
	"math/big"
	"reflect"
	"sync"

	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/phoreproject/synapse/chainhash"

	"github.com/go-interpreter/wagon/exec"

	"github.com/go-interpreter/wagon/wasm"
)

// Shard represents a single shard on the Phore blockchain.
type Shard struct {
	Module           *wasm.Module
	Storage          state.AccessInterface
	VM               *exec.VM
	ExportedFuncs    []int64
	ExecutionContext ArgumentContext

	err           error
	executionLock *sync.Mutex
}

// error informs the executor that execution has failed.
func (s *Shard) error(proc *exec.Process, err error) {
	s.err = err
	proc.Terminate()
}

// Load loads a value from state storage.
func (s *Shard) Load(proc *exec.Process, outAddr int32, inAddr int32) {
	addrHash := s.ReadHashAt(proc, inAddr)
	outHash, err := s.Storage.Get(addrHash)
	if err != nil {
		s.error(proc, err)
	}
	s.SafeWrite(proc, outAddr, outHash[:])
}

// Store stores a value to state storage.
func (s *Shard) Store(proc *exec.Process, addr int32, val int32) {
	addrHash := s.ReadHashAt(proc, addr)
	valHash := s.ReadHashAt(proc, val)
	err := s.Storage.Set(addrHash, valHash)
	if err != nil {
		panic(err)
	}
}

// LoadArgument loads an argument passed in via the transaction.
func (s *Shard) LoadArgument(proc *exec.Process, argNum int32, outAddr int32) {
	out := s.ExecutionContext.LoadArgument(argNum)
	s.SafeWrite(proc, outAddr, out)
}

// ValidateECDSA validates a certain ECDSA signature.
func (s *Shard) ValidateECDSA(proc *exec.Process, hashAddr int32, signatureAddr int32, pubkeyOut int32) int64 {
	var hash [32]byte
	s.SafeRead(proc, hashAddr, hash[:])

	var sigBytes [65]byte
	s.SafeRead(proc, signatureAddr, sigBytes[:])

	// as long as hash is a random oracle (not chosen by the user),
	// we can derive the pubkey from the signature and the hash.
	pub, _, err := secp256k1.RecoverCompact(sigBytes[:], hash[:])
	if err != nil {
		return 1
	}

	var pubkeyCompressed [33]byte

	copy(pubkeyCompressed[:], pub.Serialize())

	s.SafeWrite(proc, pubkeyOut, pubkeyCompressed[:])

	sig := DecompressSignature(sigBytes)

	if !sig.Verify(hash[:], pub) {
		return 2
	}
	return 0
}

// Hash calculates the hash of some data.
func (s *Shard) Hash(proc *exec.Process, hashOut int32, inputStart int32, inputSize int32) {
	toHash := make([]byte, inputSize)

	s.SafeRead(proc, inputStart, toHash)

	h := chainhash.HashH(toHash)

	s.SafeWrite(proc, hashOut, h[:])
}

// Log logs a message.
func (s *Shard) Log(proc *exec.Process, strPtr int32, strLen int32) {
	logMessage := make([]byte, strLen)

	s.SafeRead(proc, strPtr, logMessage)
	logrus.Debugf("log message: %s", string(logMessage))
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
func NewShard(wasmCode []byte, exportedFuncs []int64, storageAccess state.AccessInterface) (*Shard, error) {
	buf := bytes.NewBuffer(wasmCode)

	s := &Shard{
		Storage:       storageAccess,
		ExportedFuncs: exportedFuncs,
		executionLock: new(sync.Mutex),
	}

	mod, err := wasm.ReadModule(buf, func(name string) (*wasm.Module, error) {
		switch name {
		case "phore":
			m := wasm.NewModule()
			m.Types = &wasm.SectionTypes{
				Entries: []wasm.FunctionSig{
					{
						Form:        0, // value for the 'func' type constructor
						ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32},
						ReturnTypes: []wasm.ValueType{},
					},
					{
						Form:        0, // value for the 'func' type constructor
						ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
						ReturnTypes: []wasm.ValueType{wasm.ValueTypeI64},
					},
					{
						Form:       0, // value for the 'func' type constructor
						ParamTypes: []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
					},
				},
			}
			m.FunctionIndexSpace = []wasm.Function{
				{
					Sig:  &m.Types.Entries[0],
					Host: reflect.ValueOf(s.Load),
					Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
				},
				{
					Sig:  &m.Types.Entries[0],
					Host: reflect.ValueOf(s.Store),
					Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
				},
				{
					Sig:  &m.Types.Entries[1],
					Host: reflect.ValueOf(s.ValidateECDSA),
					Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
				},
				{
					Sig:  &m.Types.Entries[2],
					Host: reflect.ValueOf(s.Hash),
					Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
				},
				{
					Sig:  &m.Types.Entries[0],
					Host: reflect.ValueOf(s.LoadArgument),
					Body: &wasm.FunctionBody{},
				},
				{
					Sig:  &m.Types.Entries[0],
					Host: reflect.ValueOf(s.Log),
					Body: &wasm.FunctionBody{},
				},
			}
			m.Export = &wasm.SectionExports{
				Entries: map[string]wasm.ExportEntry{
					"load": {
						FieldStr: "load",
						Kind:     wasm.ExternalFunction,
						Index:    0,
					},
					"store": {
						FieldStr: "store",
						Kind:     wasm.ExternalFunction,
						Index:    1,
					},
					"validateECDSA": {
						FieldStr: "validateECDSA",
						Kind:     wasm.ExternalFunction,
						Index:    2,
					},
					"hash": {
						FieldStr: "hash",
						Kind:     wasm.ExternalFunction,
						Index:    3,
					},
					"loadArgument": {
						FieldStr: "loadArgument",
						Kind:     wasm.ExternalFunction,
						Index:    4,
					},
					"write_log": {
						FieldStr: "write_log",
						Kind:     wasm.ExternalFunction,
						Index:    5,
					},
				},
			}

			return m, nil
		}

		return nil, fmt.Errorf("could not find module %s", name)
	})
	if err != nil {
		return nil, err
	}

	vm, err := exec.NewVM(mod)
	if err != nil {
		return nil, err
	}

	s.VM = vm
	s.Module = mod

	return s, nil
}

// SafeRead reads from a process and handles errors while reading.
func (s *Shard) SafeRead(proc *exec.Process, location int32, out []byte) {
	_, err := proc.ReadAt(out, int64(location))
	if err != nil {
		s.error(proc, err)
	}
}

// SafeWrite writes to a process and handles errors while writing.
func (s *Shard) SafeWrite(proc *exec.Process, location int32, val []byte) {
	_, err := proc.WriteAt(val, int64(location))
	if err != nil {
		s.error(proc, err)
	}
}

// ReadHashAt reads a hash from WebAssembly memory.
func (s *Shard) ReadHashAt(proc *exec.Process, addr int32) [32]byte {
	var hash [32]byte
	s.SafeRead(proc, addr, hash[:])
	return hash
}

// RunFunc runs a shard function
func (s *Shard) RunFunc(executionContext ArgumentContext) (interface{}, error) {
	// TODO: ensure in exportedFuncs

	s.executionLock.Lock()
	defer s.executionLock.Unlock()

	fnName := executionContext.GetFunction()

	fnToCall, found := s.Module.Export.Entries[fnName]
	if !found {
		return nil, fmt.Errorf("could not find function %s", fnName)
	}

	s.ExecutionContext = executionContext
	s.err = nil

	out, err := s.VM.ExecCode(int64(fnToCall.Index))

	// set this to nil because we should re-set it every time we run a function
	s.ExecutionContext = nil

	if err != nil {
		return nil, err
	}

	if s.err != nil {
		return nil, s.err
	}

	return out, nil
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
