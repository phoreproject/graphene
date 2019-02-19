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

type attestationAssignment struct {
	slot              uint64
	shard             uint64
	committeeSize     uint64
	committeeIndex    uint64
	beaconBlockHash   chainhash.Hash
	epochBoundaryRoot chainhash.Hash
	latestCrosslinks  []primitives.Crosslink
	justifiedSlot     uint64
	justifiedRoot     chainhash.Hash
}

type proposerAssignment struct {
	slot uint64
}

type slotInformation struct {
	slot              int64
	beaconBlockHash   chainhash.Hash
	epochBoundaryRoot chainhash.Hash
	latestCrosslinks  []primitives.Crosslink
	justifiedSlot     uint64
	justifiedRoot     chainhash.Hash
	committees        []primitives.ShardAndCommittee
	proposer          uint32
}

// slotInformationFromProto gets the slot information from the protobuf format
func slotInformationFromProto(information *pb.SlotInformation) (*slotInformation, error) {
	if information.Slot < 0 {
		return &slotInformation{
			slot: -1,
		}, nil
	}

	si := &slotInformation{
		slot:          information.Slot,
		justifiedSlot: information.JustifiedSlot,
		proposer:      information.Proposer,
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

	si.committees = make([]primitives.ShardAndCommittee, len(information.Committees))
	for i := range information.Committees {
		c, err := primitives.ShardAndCommitteeFromProto(information.Committees[i])
		if err != nil {
			return nil, err
		}
		si.committees[i] = *c
	}
	return si, nil
}

// Manager is a manager that keeps track of multiple validators.
type Manager struct {
	blockchainRPC           pb.BlockchainRPCClient
	p2pRPC                  pb.P2PRPCClient
	validatorMap            map[uint32]*Validator
	keystore                Keystore
	mempool                 *mempool
	currentSlot             uint64
	config                  *config.Config
	attestationSubscription *pb.Subscription
}

// NewManager creates a new validator manager to manage some validators.
func NewManager(blockchainConn *grpc.ClientConn, p2pConn *grpc.ClientConn, validators []uint32, keystore Keystore, c *config.Config) (*Manager, error) {
	blockchainRPC := pb.NewBlockchainRPCClient(blockchainConn)
	p2pRPC := pb.NewP2PRPCClient(p2pConn)

	validatorObjs := make(map[uint32]*Validator)

	m := newMempool()

	forkDataProto, err := blockchainRPC.GetForkData(context.Background(), &empty.Empty{})
	if err != nil {
		return nil, err
	}

	forkData, err := primitives.ForkDataFromProto(forkDataProto)
	if err != nil {
		return nil, err
	}

	for i := range validators {
		validatorObjs[uint32(i)] = NewValidator(keystore, blockchainRPC, p2pRPC, validators[i], &m, c, forkData)
	}

	vm := &Manager{
		blockchainRPC: blockchainRPC,
		p2pRPC:        p2pRPC,
		validatorMap:  validatorObjs,
		keystore:      keystore,
		mempool:       &m,
		config:        c,
		currentSlot:   0,
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

		if si.slot < 0 {
			return nil
		}

		logrus.WithField("slot", b.SlotNumber).Debug("heard new slot")

		if b.SlotNumber%uint64(vm.config.EpochLength) == 0 {
			slot := b.SlotNumber - vm.config.MinAttestationInclusionDelay - vm.config.EpochLength
			if b.SlotNumber < vm.config.MinAttestationInclusionDelay+vm.config.EpochLength {
				slot = 0
			}
			vm.mempool.attestationMempool.removeAttestationsBeforeSlot(slot)
		}

		valWaitGroup := new(sync.WaitGroup)

		for _, c := range si.committees {
			for committeeIndex, validatorID := range c.Committee {
				if validator, found := vm.validatorMap[validatorID]; found {
					a := attestationAssignment{
						slot:              b.SlotNumber,
						shard:             c.Shard,
						committeeSize:     uint64(len(c.Committee)),
						committeeIndex:    uint64(committeeIndex),
						beaconBlockHash:   si.beaconBlockHash,
						epochBoundaryRoot: si.epochBoundaryRoot,
						latestCrosslinks:  si.latestCrosslinks,
						justifiedRoot:     si.justifiedRoot,
					}

					// here we should proposer just before the block and attest just after the block
					valWaitGroup.Add(1)
					go func() {
						validator.attestationRequest <- a
						valWaitGroup.Done()
					}()
				}
			}
		}

		valWaitGroup.Wait()

		if validator, found := vm.validatorMap[si.proposer]; found {
			err := validator.proposeBlock(proposerAssignment{
				slot: b.SlotNumber + 1,
			})
			if err != nil {
				return err
			}
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
	sub, err := vm.p2pRPC.Subscribe(context.Background(), &pb.SubscriptionRequest{
		Topic: "attestations",
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

	wg.Add(len(vm.validatorMap))

	for _, v := range vm.validatorMap {
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
