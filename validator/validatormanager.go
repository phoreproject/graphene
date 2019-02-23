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
	slot      uint64
	proposeAt uint64
}

type epochInformation struct {
	slots             []slotInformation
	slot              int64
	epochBoundaryRoot chainhash.Hash
	latestCrosslinks  []primitives.Crosslink
	justifiedSlot     uint64
	justifiedRoot     chainhash.Hash
}

type slotInformation struct {
	slot       int64
	committees []primitives.ShardAndCommittee
	proposeAt  uint64
}

// epochInformationFromProto gets the epoch information from the protobuf format
func epochInformationFromProto(information *pb.EpochInformation) (*epochInformation, error) {
	if information.Slot < 0 {
		return &epochInformation{
			slot: -1,
		}, nil
	}

	ei := &epochInformation{
		slot:             information.Slot,
		justifiedSlot:    information.JustifiedSlot,
		slots:            make([]slotInformation, len(information.Slots)),
		latestCrosslinks: make([]primitives.Crosslink, len(information.LatestCrosslinks)),
	}

	for i := range ei.slots {
		s, err := slotInformationFromProto(information.Slots[i])
		if err != nil {
			return nil, err
		}
		ei.slots[i] = *s
	}

	for i := range ei.latestCrosslinks {
		c, err := primitives.CrosslinkFromProto(information.LatestCrosslinks[i])
		if err != nil {
			return nil, err
		}
		ei.latestCrosslinks[i] = *c
	}

	err := ei.justifiedRoot.SetBytes(information.JustifiedHash)
	if err != nil {
		return nil, err
	}

	err = ei.epochBoundaryRoot.SetBytes(information.EpochBoundaryRoot)
	return ei, err
}

// slotInformationFromProto gets the slot information from the protobuf format
func slotInformationFromProto(information *pb.SlotInformation) (*slotInformation, error) {
	si := &slotInformation{
		slot:      information.Slot,
		proposeAt: information.ProposeAt,
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
	attestationAssignments  [][]primitives.ShardAndCommittee
	epochBoundaryRoot       chainhash.Hash
	latestCrosslinks        []primitives.Crosslink
	justifiedRoot           chainhash.Hash
	justifiedSlot           uint64
	currentSlot             uint64
	config                  *config.Config
	synced                  bool
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

	for idx, id := range validators {
		v, err := NewValidator(keystore, blockchainRPC, p2pRPC, validators[idx], &m, c, forkData)
		if err != nil {
			return nil, err
		}
		validatorObjs[uint32(id)] = v
	}

	vm := &Manager{
		blockchainRPC: blockchainRPC,
		p2pRPC:        p2pRPC,
		validatorMap:  validatorObjs,
		keystore:      keystore,
		mempool:       &m,
		config:        c,
		currentSlot:   0,
		synced:        false,
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
		if b.SlotNumber%vm.config.EpochLength == 0 || !vm.synced {
			epochInformation, err := vm.blockchainRPC.GetEpochInformation(context.Background(), &empty.Empty{})
			if err != nil {
				return err
			}

			ei, err := epochInformationFromProto(epochInformation)
			if err != nil {
				return err
			}

			if ei.slot < 0 {
				return nil
			}

			vm.attestationAssignments = make([][]primitives.ShardAndCommittee, len(ei.slots))

			for i, si := range ei.slots[vm.config.EpochLength:] {
				if si.slot == 0 || si.slot <= int64(b.SlotNumber) {
					continue
				}
				proposer := si.committees[0].Committee[(si.slot-1)%int64(len(si.committees[0].Committee))]
				if validator, found := vm.validatorMap[proposer]; found {
					go func(proposeAt uint64, slot int64) {
						validator.proposerRequest <- proposerAssignment{
							slot:      uint64(slot),
							proposeAt: proposeAt,
						}
					}(si.proposeAt, si.slot)
				}

				vm.attestationAssignments[i] = si.committees
			}

			vm.epochBoundaryRoot = ei.epochBoundaryRoot
			vm.latestCrosslinks = ei.latestCrosslinks
			vm.justifiedRoot = ei.justifiedRoot
			vm.justifiedSlot = ei.justifiedSlot
			vm.synced = true
		}

		epochIndex := b.SlotNumber % vm.config.EpochLength
		slotCommittees := vm.attestationAssignments[epochIndex]
		blockHash, err := chainhash.NewHash(b.BlockHash)
		if err != nil {
			return err
		}

		for _, committee := range slotCommittees {
			shard := committee.Shard
			for committeeIndex, vIndex := range committee.Committee {
				if validator, found := vm.validatorMap[vIndex]; found {
					att, err := validator.attestBlock(attestationAssignment{
						slot:              b.SlotNumber,
						shard:             shard,
						committeeIndex:    uint64(committeeIndex),
						committeeSize:     uint64(len(committee.Committee)),
						beaconBlockHash:   *blockHash,
						epochBoundaryRoot: vm.epochBoundaryRoot,
						latestCrosslinks:  vm.latestCrosslinks,
						justifiedRoot:     vm.justifiedRoot,
						justifiedSlot:     vm.justifiedSlot,
					})
					if err != nil {
						return err
					}

					vm.mempool.attestationMempool.processNewAttestation(*att)
				}
			}
		}

		logrus.WithField("slot", b.SlotNumber).Debug("heard new slot")

		if b.SlotNumber%uint64(vm.config.EpochLength) == 0 {
			slot := b.SlotNumber - vm.config.MinAttestationInclusionDelay - vm.config.EpochLength
			if b.SlotNumber < vm.config.MinAttestationInclusionDelay+vm.config.EpochLength {
				slot = 0
			}
			vm.mempool.attestationMempool.removeAttestationsBeforeSlot(slot)
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
