package validator

import (
	"sync"

	"github.com/phoreproject/synapse/pb"
	"google.golang.org/grpc"
)

// Manager is a manager that keeps track of multiple validators.
type Manager struct {
	rpc        *rpc.BlockchainRPCClient
	validators []*Validator
	keystore   Keystore
}

// NewManager creates a new validator manager to manage some validators.
func NewManager(conn *grpc.ClientConn, validators []uint32, keystore Keystore) (*Manager, error) {
	r := rpc.NewBlockchainRPCClient(conn)

	validatorObjs := make([]*Validator, len(validators))

	for i := range validatorObjs {
		validatorObjs[i] = NewValidator(keystore.GetKeyForValidator(validators[i]), &r, validators[i])
	}

	return &Manager{
		rpc:        &r,
		validators: validatorObjs,
		keystore:   keystore,
	}, nil
}

// Start starts goroutines for each validator
func (vm *Manager) Start() {
	var wg sync.WaitGroup

	wg.Add(len(vm.validators))

	for _, v := range vm.validators {
		vClosed := v
		go func() {
			defer wg.Done()
			vClosed.RunValidator()
		}()
	}

	wg.Wait()
}
