package validator

import (
	"context"
	"sync"
	"time"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/pb"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

// Notifier handles notifications for each validator.
type Notifier struct {
	newSlot  chan slotInformation
	newCycle chan bool
}

// NewNotifier initializes a new validator notifier.
func NewNotifier() *Notifier {
	return &Notifier{newSlot: make(chan slotInformation), newCycle: make(chan bool)}
}

type slotInformation struct {
	slot              uint64
	beaconBlockHash   chainhash.Hash
	epochBoundaryRoot chainhash.Hash
	latestCrosslinks  []primitives.Crosslink
	justifiedSlot     uint64
	justifiedRoot     chainhash.Hash
}

// slotInformationFromProto gets the slot information from the protobuf format
func slotInformationFromProto(information *pb.SlotInformation) (*slotInformation, error) {
	si := &slotInformation{
		slot:          information.Slot,
		justifiedSlot: information.JustifiedSlot,
	}

	err := si.beaconBlockHash.SetBytes(information.BeaconBlockHash)
	if err != nil {
		return nil, err
	}

	err = si.epochBoundaryRoot.SetBytes(information.EpochBoundaryRoot)
	if err != nil {
		return nil, err
	}

	err = si.justifiedRoot.SetBytes(information.JustifiedRoot)
	if err != nil {
		return nil, err
	}

	si.latestCrosslinks = make([]primitives.Crosslink, len(information.LatestCrosslinks))
	for i := range si.latestCrosslinks {
		cl, err := primitives.CrosslinkFromProto(information.LatestCrosslinks[i])
		if err != nil {
			return nil, err
		}
		si.latestCrosslinks[i] = *cl
	}
	return si, nil
}

// SendNewSlot notifies the validator of a new slot.
func (n *Notifier) SendNewSlot(slot slotInformation) {
	n.newSlot <- slot
}

// SendNewCycle notifies the validator of a new cycle.
func (n *Notifier) SendNewCycle() {
	n.newCycle <- true
}

// Manager is a manager that keeps track of multiple validators.
type Manager struct {
	blockchainRPC pb.BlockchainRPCClient
	p2pRPC        pb.P2PRPCClient
	validators    []*Validator
	keystore      Keystore
	notifiers     []*Notifier
}

// NewManager creates a new validator manager to manage some validators.
func NewManager(blockchainConn *grpc.ClientConn, p2pConn *grpc.ClientConn, validators []uint32, keystore Keystore) (*Manager, error) {
	blockchainRPC := pb.NewBlockchainRPCClient(blockchainConn)
	p2pRPC := pb.NewP2PRPCClient(p2pConn)

	validatorObjs := make([]*Validator, len(validators))

	notifiers := make([]*Notifier, len(validators))
	for i := range notifiers {
		notifiers[i] = NewNotifier()
	}

	for i := range validatorObjs {
		validatorObjs[i] = NewValidator(keystore.GetKeyForValidator(validators[i]), blockchainRPC, p2pRPC, validators[i], notifiers[i].newSlot, notifiers[i].newCycle)
	}

	return &Manager{
		blockchainRPC: blockchainRPC,
		p2pRPC:        p2pRPC,
		validators:    validatorObjs,
		keystore:      keystore,
		notifiers:     notifiers,
	}, nil
}

// ListenForBlockAndCycle listens for any new blocks or cycles and relays
// the information to validators.
func (vm *Manager) ListenForBlockAndCycle() error {
	t := time.NewTicker(time.Second)

	currentSlot := int64(-1)

	for {
		<-t.C

		b, err := vm.blockchainRPC.GetSlotNumber(context.Background(), &empty.Empty{})
		if err != nil {
			return err
		}

		if currentSlot == int64(b.SlotNumber) {
			continue
		}

		siProto, err := vm.blockchainRPC.GetSlotInformation(context.Background(), &empty.Empty{})
		if err != nil {
			return err
		}

		si, err := slotInformationFromProto(siProto)
		if err != nil {
			return err
		}

		logrus.WithField("slot", b.SlotNumber).Debug("heard new slot")

		currentSlot = int64(b.SlotNumber)

		newCycle := false

		if b.SlotNumber%uint64(config.MainNetConfig.EpochLength) == 0 {
			newCycle = true
		}

		for _, n := range vm.notifiers {
			if newCycle {
				go n.SendNewCycle()
			}
			go n.SendNewSlot(*si)
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
