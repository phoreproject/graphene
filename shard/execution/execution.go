package execution

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"

	"github.com/decred/dcrd/dcrec/secp256k1"

	"github.com/go-interpreter/wagon/exec"

	"github.com/go-interpreter/wagon/wasm"
)

// Shard represents a single shard on the Phore blockchain.
type Shard struct {
	Module        *wasm.Module
	Storage       Storage
	VM            *exec.VM
	ExportedFuncs []int64
}

// DecompressSignature decompresses a signature.
func DecompressSignature(sig [65]byte) *secp256k1.Signature {
	R := new(big.Int)
	S := new(big.Int)

	R.SetBytes(sig[1:33])
	S.SetBytes(sig[33:65])

	return secp256k1.NewSignature(R, S)
}

// NewShard creates a new shard given some WASM code, exported funcs, and storage backend.
func NewShard(wasmCode []byte, exportedFuncs []int64, storage Storage) (*Shard, error) {
	buf := bytes.NewBuffer(wasmCode)
	mod, err := wasm.ReadModule(buf, func(name string) (*wasm.Module, error) {
		switch name {
		case "phore":
			load := func(proc *exec.Process, addr int64) int64 {
				return storage.PhoreLoad(addr)
			}

			store := func(proc *exec.Process, addr int64, val int64) {
				storage.PhoreStore(addr, val)
			}

			// validate ECDSA signature
			// ECDSA signatures in Phore use the following format:
			//
			validateECDSA := func(proc *exec.Process, hashAddr int32, signatureAddr int32, pubkeyOut int32) int64 {
				var hash [32]byte
				_, err := proc.ReadAt(hash[:], int64(hashAddr))
				if err != nil {
					fmt.Println("error reading hash from ECDSA")
					proc.Terminate()
				}

				var sigBytes [65]byte
				_, err = proc.ReadAt(sigBytes[:], int64(signatureAddr))
				if err != nil {
					fmt.Println("error reading signature from ECDSA")
					proc.Terminate()
				}

				// as long as hash is a random oracle (not chosen by the user),
				// we can derive the pubkey from the signature and the hash.
				pub, _, err := secp256k1.RecoverCompact(sigBytes[:], hash[:])
				if err != nil {
					fmt.Printf("error recovering public key from ECDSA: %s\n", err.Error())
					return 0
				}

				var pubkeyCompressed [33]byte

				copy(pubkeyCompressed[:], pub.Serialize())

				_, err = proc.WriteAt(pubkeyCompressed[:], int64(pubkeyOut))
				if err != nil {
					fmt.Println("error writing pubkey from ECDSA")
					proc.Terminate()
				}

				sig := DecompressSignature(sigBytes)

				if sig.Verify(hash[:], pub) {
					return 1
				}
				return 0
			}

			m := wasm.NewModule()
			m.Types = &wasm.SectionTypes{
				Entries: []wasm.FunctionSig{
					{
						Form:        0, // value for the 'func' type constructor
						ParamTypes:  []wasm.ValueType{wasm.ValueTypeI64},
						ReturnTypes: []wasm.ValueType{wasm.ValueTypeI64},
					},
					{
						Form:       0, // value for the 'func' type constructor
						ParamTypes: []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64},
					},
					{
						Form:        0, // value for the 'func' type constructor
						ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
						ReturnTypes: []wasm.ValueType{wasm.ValueTypeI64},
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
					Sig:  &m.Types.Entries[1],
					Host: reflect.ValueOf(store),
					Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
				},
				{
					Sig:  &m.Types.Entries[2],
					Host: reflect.ValueOf(validateECDSA),
					Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
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
		storage,
		vm,
		exportedFuncs,
	}, nil
}

// RunFunc runs a shard function
func (s *Shard) RunFunc(fnIndex int64, args ...uint64) (interface{}, error) {
	// TODO: ensure in exportedFuncs
	out, err := s.VM.ExecCode(fnIndex, args...)
	if err != nil {
		return nil, err
	}

	return out, nil

}

// Storage is the storage backend interface
type Storage interface {
	PhoreLoad(address int64) int64
	PhoreStore(address int64, value int64)
}

// MemoryStorage is a memory-backed storage backend
type MemoryStorage struct {
	memory map[int64]int64
}

// NewMemoryStorage creates a new memory-backed storage backend
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		memory: make(map[int64]int64),
	}
}

// PhoreLoad loads a value from memory.
func (m *MemoryStorage) PhoreLoad(address int64) int64 {
	if value, found := m.memory[address]; found {
		return value
	}
	return 0
}

// PhoreStore stores a value to memory.
func (m *MemoryStorage) PhoreStore(address int64, val int64) {
	m.memory[address] = val
}
