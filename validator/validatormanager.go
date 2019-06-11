package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/phoreproject/synapse/utils"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/pb"
)

type attestationAssignment struct {
	slot             uint64
	shard            uint64
	committeeSize    uint64
	committeeIndex   uint64
	beaconBlockHash  chainhash.Hash
	latestCrosslinks []primitives.Crosslink
	sourceEpoch      uint64
	sourceHash       chainhash.Hash
	targetEpoch      uint64
	targetHash       chainhash.Hash
}

type proposerAssignment struct {
	slot uint64
}

type epochInformation struct {
	slots                  []slotInformation
	slot                   int64
	targetHash             chainhash.Hash
	justifiedEpoch         uint64
	latestCrosslinks       []primitives.Crosslink
	previousCrosslinks     []primitives.Crosslink
	justifiedHash          chainhash.Hash
	epochIndex             uint64
	previousTargetHash     chainhash.Hash
	previousJustifiedEpoch uint64
	previousJustifiedHash  chainhash.Hash
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
		slots:                  make([]slotInformation, len(information.Slots)),
		slot:                   information.Slot,
		justifiedEpoch:         information.JustifiedEpoch,
		previousJustifiedEpoch: information.PreviousJustifiedEpoch,
		epochIndex:             information.EpochIndex,
		latestCrosslinks:       make([]primitives.Crosslink, len(information.LatestCrosslinks)),
		previousCrosslinks:     make([]primitives.Crosslink, len(information.PreviousCrosslinks)),
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

	for i := range ei.previousCrosslinks {
		c, err := primitives.CrosslinkFromProto(information.PreviousCrosslinks[i])
		if err != nil {
			return nil, err
		}
		ei.previousCrosslinks[i] = *c
	}

	err := ei.targetHash.SetBytes(information.TargetHash)
	if err != nil {
		return nil, err
	}

	err = ei.justifiedHash.SetBytes(information.JustifiedHash)
	if err != nil {
		return nil, err
	}

	err = ei.previousTargetHash.SetBytes(information.PreviousTargetHash)
	if err != nil {
		return nil, err
	}

	err = ei.previousJustifiedHash.SetBytes(information.PreviousJustifiedHash)
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
	latestEpochInformation epochInformation
	currentSlot            uint64
	config                 *config.Config
	synced                 bool
}

// NewManager creates a new validator manager to manage some validators.
func NewManager(ctx context.Context, blockchainRPC pb.BlockchainRPCClient, validators []uint32, keystore Keystore, c *config.Config) (*Manager, error) {
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
		v, err := NewValidator(ctx, keystore, blockchainRPC, validators[idx], c, forkData)
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

// UpdateEpochInformation updates epoch information from the beacon chain
func (vm *Manager) UpdateEpochInformation(slotNumber uint64) error {
	epochInformation, err := vm.blockchainRPC.GetEpochInformation(context.Background(), &pb.EpochInformationRequest{EpochIndex: slotNumber / vm.config.EpochLength})
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

	for i, si := range ei.slots {
		vm.attestationAssignments[i] = si.committees
	}

	vm.latestEpochInformation = *ei
	vm.synced = true

	return nil
}

// NewSlot is run when a new slot starts.
func (vm *Manager) NewSlot(slotNumber uint64) error {
	stateSlot := vm.latestEpochInformation.epochIndex * vm.config.EpochLength
	earliestSlot := int64(stateSlot) - int64(vm.config.EpochLength)

	logrus.WithField("slot", slotNumber).Debug("heard new slot")

	proposerSlotCommittees := vm.attestationAssignments[int64(slotNumber-1)-earliestSlot]

	proposer := proposerSlotCommittees[0].Committee[(slotNumber-1)%uint64(len(proposerSlotCommittees[0].Committee))]
	if validator, found := vm.validatorMap[proposer]; found {
		err := validator.proposeBlock(context.Background(), proposerAssignment{
			slot: uint64(slotNumber),
		})
		if err != nil {
			fmt.Println(err)
		}
	}

	<-time.After(time.Millisecond * 500 * time.Duration(vm.config.SlotDuration))

	logrus.WithField("slot", slotNumber).Debug("requesting epoch information")
	if err := vm.UpdateEpochInformation(slotNumber); err != nil {
		return err
	}
	logrus.WithField("index", vm.latestEpochInformation.epochIndex).Debug("got epoch information")

	stateSlot = vm.latestEpochInformation.epochIndex * vm.config.EpochLength
	earliestSlot = int64(stateSlot) - int64(vm.config.EpochLength)

	slotToAttest := slotNumber

	epochIndex := int64(slotToAttest) - earliestSlot

	slotCommittees := vm.attestationAssignments[epochIndex-1] // we actually want to attest MinAttestationInclusionDistance after the slot

	blockHashResponse, err := vm.blockchainRPC.GetBlockHash(context.Background(), &pb.GetBlockHashRequest{
		SlotNumber: slotToAttest,
	})
	if err != nil {
		return err
	}

	blockHash, err := chainhash.NewHash(blockHashResponse.Hash)
	if err != nil {
		return err
	}

	if slotToAttest > 0 {
		for _, committee := range slotCommittees {
			shard := committee.Shard

			for committeeIndex, vIndex := range committee.Committee {
				if validator, found := vm.validatorMap[vIndex]; found {
					sourceEpoch := vm.latestEpochInformation.justifiedEpoch
					sourceHash := vm.latestEpochInformation.justifiedHash
					targetEpoch := vm.latestEpochInformation.epochIndex
					targetHash := vm.latestEpochInformation.targetHash
					crosslinks := vm.latestEpochInformation.latestCrosslinks

					if slotToAttest%vm.config.EpochLength == 0 {
						targetEpoch--
						targetHash = vm.latestEpochInformation.previousTargetHash
						sourceEpoch = vm.latestEpochInformation.previousJustifiedEpoch
						sourceHash = vm.latestEpochInformation.previousJustifiedHash
						crosslinks = vm.latestEpochInformation.previousCrosslinks
					}

					att, err := validator.attestBlock(attestationAssignment{
						slot:             slotToAttest,
						shard:            shard,
						committeeIndex:   uint64(committeeIndex),
						committeeSize:    uint64(len(committee.Committee)),
						beaconBlockHash:  *blockHash,
						latestCrosslinks: crosslinks,
						sourceEpoch:      sourceEpoch,
						sourceHash:       sourceHash,
						targetEpoch:      targetEpoch,
						targetHash:       targetHash,
					})
					if err != nil {
						return err
					}

					_, err = vm.blockchainRPC.SubmitAttestation(context.Background(), att.ToProto())
					if err != nil {
						fmt.Println(err)
						return nil
					}
				}
			}
		}
	}

	return nil
}

// ListenForBlockAndCycle listens for any new blocks or cycles and relays
// the information to validators.
func (vm *Manager) ListenForBlockAndCycle() error {
	stateProto, err := vm.blockchainRPC.GetState(context.Background(), &empty.Empty{})
	if err != nil {
		return err
	}

	state, err := primitives.StateFromProto(stateProto.State)
	if err != nil {
		return err
	}

	slotNumberResponse, err := vm.blockchainRPC.GetSlotNumber(context.Background(), &empty.Empty{})
	if err != nil {
		return err
	}

	currentSlot := slotNumberResponse.SlotNumber

	nextEpochSlot := currentSlot - currentSlot%vm.config.EpochLength + 1

	logrus.WithField("slot", currentSlot).WithField("epochSlot", nextEpochSlot).Debug("waiting for next epoch")

	genesisTime := state.GenesisTime

	nextSlotTime := time.Unix(int64(nextEpochSlot*uint64(vm.config.SlotDuration)+genesisTime), 5e8)
	slotNumber := nextEpochSlot

	<-time.NewTimer(nextSlotTime.Sub(utils.Now())).C

	logrus.WithField("slot", slotNumber).Debug("requesting epoch information")
	if err := vm.UpdateEpochInformation(slotNumber); err != nil {
		return err
	}
	logrus.WithField("index", vm.latestEpochInformation.epochIndex).Debug("got epoch information")

	for {
		err := vm.NewSlot(slotNumber)
		if err != nil {
			return err
		}

		slotNumber = slotNumber + 1
		nextSlotTime = time.Unix(int64(slotNumber*uint64(vm.config.SlotDuration)+genesisTime), 5e8)

		<-time.NewTimer(nextSlotTime.Sub(utils.Now())).C
	}

}

// Start starts goroutines for each validator
func (vm *Manager) Start() error {
	return vm.ListenForBlockAndCycle()
}
