package validator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/pb"

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
	blockchainRPC           pb.BlockchainRPCClient
	p2pRPC                  pb.P2PRPCClient
	validators              []*Validator
	keystore                Keystore
	notifiers               []*Notifier
	mempool                 *mempool
	currentSlot             uint64
	config                  *config.Config
	attestationSubscription *pb.Subscription
}

// NewManager creates a new validator manager to manage some validators.
func NewManager(blockchainConn *grpc.ClientConn, p2pConn *grpc.ClientConn, validators []uint32, keystore Keystore, c *config.Config) (*Manager, error) {
	blockchainRPC := pb.NewBlockchainRPCClient(blockchainConn)
	p2pRPC := pb.NewP2PRPCClient(p2pConn)

	validatorObjs := make([]*Validator, len(validators))

	notifiers := make([]*Notifier, len(validators))
	for i := range notifiers {
		notifiers[i] = NewNotifier()
	}

	m := newMempool()

	for i := range validatorObjs {
		validatorObjs[i] = NewValidator(keystore, blockchainRPC, p2pRPC, validators[i], notifiers[i].newSlot, notifiers[i].newCycle, &m, c)
	}

	vm := &Manager{
		blockchainRPC: blockchainRPC,
		p2pRPC:        p2pRPC,
		validators:    validatorObjs,
		keystore:      keystore,
		notifiers:     notifiers,
		mempool:       &m,
		config:        c,
		currentSlot:   0,
	}

	logrus.Debug("cancelling old attestation listener")
	err := vm.CancelAttestationsListener()
	if err != nil {
		return nil, err
	}
	logrus.Debug("initializing attestation listener")
	err = vm.ListenForNewAttestations()
	if err != nil {
		return nil, err
	}

	return vm, nil
}

// UpdateSlotNumber gets the slot number from RPC and runs validator actions as needed.
func (vm *Manager) UpdateSlotNumber() error {
	b, err := vm.blockchainRPC.GetSlotNumber(context.Background(), &empty.Empty{})
	if err != nil {
		return err
	}

	if vm.currentSlot != b.SlotNumber+1 {
		siProto, err := vm.blockchainRPC.GetSlotInformation(context.Background(), &empty.Empty{})
		if err != nil {
			return err
		}

		si, err := slotInformationFromProto(siProto)
		if err != nil {
			return err
		}

		logrus.WithField("slot", b.SlotNumber).Debug("heard new slot")

		newCycle := false

		if b.SlotNumber%uint64(vm.config.EpochLength) == 0 && b.SlotNumber != 0 {
			newCycle = true
		}

		if newCycle {
			logrus.Debug("cancelling old attestation listener")
			err := vm.CancelAttestationsListener()
			if err != nil {
				return err
			}
			logrus.Debug("initializing attestation listener")
			err = vm.ListenForNewAttestations()
			if err != nil {
				return err
			}
			logrus.Debug("done")
		}

		for _, n := range vm.notifiers {
			if newCycle {
				go n.SendNewCycle()
			}
			go n.SendNewSlot(*si)
		}

		vm.currentSlot = b.SlotNumber + 1
	}

	return nil
}

// CancelAttestationsListener cancels the old subscription.
func (vm *Manager) CancelAttestationsListener() error {
	if vm.attestationSubscription != nil {
		_, err := vm.p2pRPC.Unsubscribe(context.Background(), vm.attestationSubscription)
		return err
	}
	return nil
}

// ListenForNewAttestations listens for new attestations from this epoch.
func (vm *Manager) ListenForNewAttestations() error {
	epochNum := vm.currentSlot / vm.config.EpochLength

	topic := fmt.Sprintf("attestations epoch %d", epochNum/vm.config.EpochLength)

	sub, err := vm.p2pRPC.Subscribe(context.Background(), &pb.SubscriptionRequest{
		Topic: topic,
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	logrus.Debug("starting listener")

	listener, err := vm.p2pRPC.ListenForMessages(context.Background(), sub)
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, err := listener.Recv()
			if err != nil {
				return
			}

			attestationProto := &pb.Attestation{}
			err = proto.Unmarshal(msg.Data, attestationProto)
			if err != nil {
				continue
			}

			attestation, err := primitives.AttestationFromProto(attestationProto)
			if err != nil {
				continue
			}

			// do some checks to make sure the attestation is valid
			vm.mempool.attestationMempool.processNewAttestation(*attestation)
		}
	}()

	vm.attestationSubscription = sub

	return nil
}

// ListenForBlockAndCycle listens for any new blocks or cycles and relays
// the information to validators.
func (vm *Manager) ListenForBlockAndCycle() error {
	t := time.NewTicker(time.Second)

	for {
		<-t.C

		err := vm.UpdateSlotNumber()
		if err != nil {
			return err
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
