package app

import (
	"context"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/validator"
)

// ValidatorApp is the app to run the validator runtime.
type ValidatorApp struct {
	config ValidatorConfig
	ctx    context.Context
	cancel context.CancelFunc
	err    chan error
}

// NewValidatorApp creates a new validator app from the config.
func NewValidatorApp(config ValidatorConfig) *ValidatorApp {
	ctx, cancel := context.WithCancel(context.Background())
	return &ValidatorApp{
		config: config,
		ctx:    ctx,
		cancel: cancel,
		err:    make(chan error),
	}
}

// Run starts the validator app.
func (v *ValidatorApp) Run() error {
	vm, err := validator.NewManager(v.ctx, v.config.BlockchainConn, v.config.ValidatorIndices, validator.NewRootKeyStore(v.config.RootKey), &config.MainNetConfig)
	if err != nil {
		return err
	}

	go func() {
		err := vm.Start()
		if err != nil {
			v.err <- err
		}
	}()

	return nil
}

// Exit exits the validator app.
func (v *ValidatorApp) Exit() {
	v.cancel()
}
