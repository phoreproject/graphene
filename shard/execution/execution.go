package execution

import (
	"bytes"
	"fmt"
	"reflect"

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

// NewShard creates a new shard given some WASM code, exported funcs, and storage backend.
func NewShard(wasmCode []byte, exportedFuncs []int64, storage Storage) (*Shard, error) {
	buf := bytes.NewBuffer(wasmCode)
	mod, err := wasm.ReadModule(buf, func(name string) (*wasm.Module, error) {
		fmt.Println(name)
		switch name {
		case "phore":
			load := func(proc *exec.Process, addr int64) int64 {
				return storage.PhoreLoad(addr)
			}

			store := func(proc *exec.Process, addr int64, val int64) {
				storage.PhoreStore(addr, val)
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
