package execution

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/phoreproject/synapse/shard/state"
	"math/big"
	"reflect"

	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/phoreproject/synapse/chainhash"

	"github.com/go-interpreter/wagon/exec"

	"github.com/go-interpreter/wagon/wasm"
)

// ArgumentContext represents a single execution of a shard.
type ArgumentContext interface {
	LoadArgument(argumentNumber int32) []byte
}

// EmptyContext is a context that doesn't resolve arguments.
type EmptyContext struct{}

// LoadArgument for empty context doesn't load anything.
func (e EmptyContext) LoadArgument(argumentNumber int32) []byte { return nil }

// Shard represents a single shard on the Phore blockchain.
type Shard struct {
	Module           *wasm.Module
	Storage          state.AccessInterface
	VM               *exec.VM
	ExportedFuncs    []int64
	ExecutionContext ArgumentContext
}

// DecompressSignature decompresses a signature.
func DecompressSignature(sig [65]byte) *secp256k1.Signature {
	R := new(big.Int)
	S := new(big.Int)

	R.SetBytes(sig[1:33])
	S.SetBytes(sig[33:65])

	return secp256k1.NewSignature(R, S)
}

// SafeRead reads from a process and handles errors while reading.
func SafeRead(proc *exec.Process, location int32, out []byte) {
	_, err := proc.ReadAt(out, int64(location))
	if err != nil {
		fmt.Printf("error reading from memory at address: 0x%x\n", location)
		proc.Terminate()
	}
}

// SafeWrite writes to a process and handles errors while writing.
func SafeWrite(proc *exec.Process, location int32, val []byte) {
	_, err := proc.WriteAt(val, int64(location))
	if err != nil {
		fmt.Printf("error writing to memory at address: 0x%x\n", location)
		proc.Terminate()
	}
}

// ReadHashAt reads a hash from WebAssembly memory.
func ReadHashAt(proc *exec.Process, addr int32) [32]byte {
	var hash [32]byte
	SafeRead(proc, addr, hash[:])
	return hash
}

// NewShard creates a new shard given some WASM code, exported funcs, and storage backend.
func NewShard(wasmCode []byte, exportedFuncs []int64, storageAccess state.AccessInterface, execContext ArgumentContext) (*Shard, error) {
	buf := bytes.NewBuffer(wasmCode)
	mod, err := wasm.ReadModule(buf, func(name string) (*wasm.Module, error) {
		switch name {
		case "phore":
			load := func(proc *exec.Process, outAddr int32, inAddr int32) {
				addrHash := ReadHashAt(proc, inAddr)
				outHash, err := storageAccess.Get(addrHash)
				if err != nil {
					// this should be handled better
					panic(err)
				}
				//fmt.Printf("load: load %x, got %x\n", addrHash[:], outHash[:])
				SafeWrite(proc, outAddr, outHash[:])
			}

			store := func(proc *exec.Process, addr int32, val int32) {
				addrHash := ReadHashAt(proc, addr)
				valHash := ReadHashAt(proc, val)
				err := storageAccess.Set(addrHash, valHash)
				if err != nil {
					panic(err)
				}
				//fmt.Printf("store: set addr %x to %x\n", addrHash[:], valHash[:])
			}

			loadArgument := func(proc *exec.Process, argNum int32, outAddr int32) {
				out := execContext.LoadArgument(argNum)
				//fmt.Printf("loadArg: get arg %d, got %x\n", argNum, out)
				SafeWrite(proc, outAddr, out)
			}

			// validate ECDSA signature
			// ECDSA signatures in Phore use the following format:
			//
			validateECDSA := func(proc *exec.Process, hashAddr int32, signatureAddr int32, pubkeyOut int32) int64 {
				var hash [32]byte
				SafeRead(proc, hashAddr, hash[:])

				var sigBytes [65]byte
				SafeRead(proc, signatureAddr, sigBytes[:])

				//fmt.Printf("validateECDSA: hash: %x, sig: %x\n", hash, sigBytes)

				// as long as hash is a random oracle (not chosen by the user),
				// we can derive the pubkey from the signature and the hash.
				pub, _, err := secp256k1.RecoverCompact(sigBytes[:], hash[:])
				if err != nil {
					return 1
				}

				var pubkeyCompressed [33]byte

				copy(pubkeyCompressed[:], pub.Serialize())

				SafeWrite(proc, pubkeyOut, pubkeyCompressed[:])

				sig := DecompressSignature(sigBytes)

				if !sig.Verify(hash[:], pub) {
					return 2
				}
				return 0
			}

			hash := func(proc *exec.Process, hashOut int32, inputStart int32, inputSize int32) {
				toHash := make([]byte, inputSize)

				SafeRead(proc, inputStart, toHash)

				h := chainhash.HashH(toHash)

				//fmt.Printf("hash: hashing %x to get %x\n", toHash, h.CloneBytes())

				SafeWrite(proc, hashOut, h[:])
			}

			writeLog := func(proc *exec.Process, strPtr int32, strlen int32) {
				logMessage := make([]byte, strlen)

				SafeRead(proc, strPtr, logMessage)
				fmt.Printf("log message: %s\n", string(logMessage))
			}

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
					Host: reflect.ValueOf(load),
					Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
				},
				{
					Sig:  &m.Types.Entries[0],
					Host: reflect.ValueOf(store),
					Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
				},
				{
					Sig:  &m.Types.Entries[1],
					Host: reflect.ValueOf(validateECDSA),
					Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
				},
				{
					Sig:  &m.Types.Entries[2],
					Host: reflect.ValueOf(hash),
					Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
				},
				{
					Sig:  &m.Types.Entries[0],
					Host: reflect.ValueOf(loadArgument),
					Body: &wasm.FunctionBody{},
				},
				{
					Sig:  &m.Types.Entries[0],
					Host: reflect.ValueOf(writeLog),
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

	return &Shard{
		mod,
		storageAccess,
		vm,
		exportedFuncs,
		execContext,
	}, nil
}

// RunFunc runs a shard function
func (s *Shard) RunFunc(fnName string) (interface{}, error) {
	// TODO: ensure in exportedFuncs

	fnToCall, found := s.Module.Export.Entries[fnName]
	if !found {
		return nil, fmt.Errorf("could not find function %s", fnName)
	}

	out, err := s.VM.ExecCode(int64(fnToCall.Index))
	if err != nil {
		return nil, err
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
