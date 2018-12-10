package validator

import (
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/pb"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

// Notifier handles notifications for each validator.
type Notifier struct {
	newSlot  chan uint64
	newCycle chan bool
}

// NewNotifier initializes a new validator notifier.
func NewNotifier() *Notifier {
	return &Notifier{newSlot: make(chan uint64), newCycle: make(chan bool)}
}

// SendNewSlot notifies the validator of a new slot.
func (n *Notifier) SendNewSlot(slot uint64) {
	n.newSlot <- slot
}

// SendNewCycle notifies the validator of a new cycle.
func (n *Notifier) SendNewCycle() {
	n.newCycle <- true
}

// Manager is a manager that keeps track of multiple validators.
type Manager struct {
	rpc        pb.BlockchainRPCClient
	p2p        pb.P2PRPCClient
	validators []*Validator
	keystore   Keystore
	notifiers  []*Notifier
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewManager creates a new validator manager to manage some validators.
func NewManager(connBeacon *grpc.ClientConn, connP2P *grpc.ClientConn, validators []uint32, keystore Keystore) (*Manager, error) {
	beaconRPC := pb.NewBlockchainRPCClient(connBeacon)

	p2pRPC := pb.NewP2PRPCClient(connP2P)

	validatorObjs := make([]*Validator, len(validators))

	notifiers := make([]*Notifier, len(validators))
	for i := range notifiers {
		notifiers[i] = NewNotifier()
	}

	ctx, cancel := context.WithCancel(context.Background())

	for i := range validatorObjs {
		validatorObjs[i] = NewValidator(ctx, keystore.GetKeyForValidator(validators[i]), beaconRPC, p2pRPC, validators[i], notifiers[i].newSlot, notifiers[i].newCycle)
	}

	return &Manager{
		rpc:        beaconRPC,
		p2p:        p2pRPC,
		validators: validatorObjs,
		keystore:   keystore,
		notifiers:  notifiers,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// ListenForBlockAndCycle listens for any new blocks or cycles and relays
// the information to validators.
func (vm *Manager) ListenForBlockAndCycle() error {
	t := time.NewTicker(time.Second)

	currentSlot := int64(-1)

	for {
		<-t.C

		b, err := vm.rpc.GetSlotNumber(context.Background(), &empty.Empty{})
		if err != nil {
			return err
		}

		if currentSlot == int64(b.SlotNumber) {
			continue
		}

		logrus.WithField("slot", b.SlotNumber).Debug("heard new slot")

		currentSlot = int64(b.SlotNumber)

		newCycle := false

		if b.SlotNumber%uint64(beacon.MainNetConfig.CycleLength) == 0 {
			newCycle = true
		}

		for _, n := range vm.notifiers {
			if newCycle {
				go n.SendNewCycle()
			}
			go n.SendNewSlot(b.SlotNumber)
		}
	}
}

// Start starts goroutines for each validator
func (vm *Manager) Start() {
	go func() {
		err := vm.ListenForBlockAndCycle()
		if err != nil {
			panic(err)
		}
	}()

	var wg sync.WaitGroup

	wg.Add(len(vm.validators))

	for _, v := range vm.validators {
		vClosed := v
		go func() {
			defer wg.Done()
			err := vClosed.RunValidator()
			if err != nil {
				panic(err)
			}
		}()
	}

	wg.Wait()
}
