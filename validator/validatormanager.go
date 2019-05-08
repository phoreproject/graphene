package validator

import (
	"context"
	"sync"
	"time"

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
	blockchainRPC          pb.BlockchainRPCClient
	validatorMap           map[uint32]*Validator
	keystore               Keystore
	attestationAssignments [][]primitives.ShardAndCommittee
	epochBoundaryRoot      chainhash.Hash
	latestCrosslinks       []primitives.Crosslink
	justifiedRoot          chainhash.Hash
	justifiedSlot          uint64
	currentSlot            uint64
	config                 *config.Config
	synced                 bool
}

// NewManager creates a new validator manager to manage some validators.
func NewManager(blockchainConn *grpc.ClientConn, validators []uint32, keystore Keystore, c *config.Config) (*Manager, error) {
	blockchainRPC := pb.NewBlockchainRPCClient(blockchainConn)

	validatorObjs := make(map[uint32]*Validator)

	forkDataProto, err := blockchainRPC.GetForkData(context.Background(), &empty.Empty{})
	if err != nil {
		return nil, err
	}

	forkData, err := primitives.ForkDataFromProto(forkDataProto)
	if err != nil {
		return nil, err
	}

	for idx, id := range validators {
		v, err := NewValidator(keystore, blockchainRPC, validators[idx], c, forkData)
		if err != nil {
			return nil, err
		}
		validatorObjs[uint32(id)] = v
	}

	vm := &Manager{
		blockchainRPC: blockchainRPC,
		validatorMap:  validatorObjs,
		keystore:      keystore,
		config:        c,
		currentSlot:   0,
		synced:        false,
	}
	logrus.Debug("initializing attestation listener")

	return vm, nil
}

// UpdateSlotNumber gets the slot number from RPC and runs validator actions as needed.
func (vm *Manager) UpdateSlotNumber() (bool, error) {
	b, err := vm.blockchainRPC.GetSlotNumber(context.Background(), &empty.Empty{})
	if err != nil {
		return true, err
	}

	if vm.currentSlot != b.SlotNumber+1 {
		if b.SlotNumber%vm.config.EpochLength == 0 || !vm.synced {
			epochInformation, err := vm.blockchainRPC.GetEpochInformation(context.Background(), &empty.Empty{})
			if err != nil {
				return true, err
			}

			ei, err := epochInformationFromProto(epochInformation)
			if err != nil {
				return false, err
			}

			if ei.slot < 0 {
				return true, nil
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
			return false, err
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
						return false, err
					}

					_, err = vm.blockchainRPC.SubmitAttestation(context.Background(), att.ToProto())
					if err != nil {
						return false, err
					}
				}
			}
		}

		logrus.WithField("slot", b.SlotNumber).Debug("heard new slot")

		vm.currentSlot = b.SlotNumber + 1
	}

	return true, nil
}

// ListenForBlockAndCycle listens for any new blocks or cycles and relays
// the information to validators.
func (vm *Manager) ListenForBlockAndCycle() error {
	t := time.NewTicker(time.Second)

	for {
		<-t.C

		canContinue, err := vm.UpdateSlotNumber()
		if err != nil {
			if !canContinue {
				return err
			}
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
