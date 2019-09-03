package validator

import (
	"context"
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
	slots                  [][]primitives.ShardAndCommittee
	slotsNextShuffling     [][]primitives.ShardAndCommittee
	earliestSlot           int64
	targetHash             chainhash.Hash
	justifiedEpoch         uint64
	latestCrosslinks       []primitives.Crosslink
	previousCrosslinks     []primitives.Crosslink
	justifiedHash          chainhash.Hash
	previousTargetHash     chainhash.Hash
	previousJustifiedEpoch uint64
	previousJustifiedHash  chainhash.Hash
}

// epochInformationFromProto gets the epoch information from the protobuf format
func epochInformationFromProto(information *pb.EpochInformation) (*epochInformation, error) {
	ei := &epochInformation{
		slots:                  make([][]primitives.ShardAndCommittee, len(information.ShardCommitteesForSlots)),
		slotsNextShuffling:     make([][]primitives.ShardAndCommittee, len(information.ShardCommitteesForNextEpoch)),
		justifiedEpoch:         information.JustifiedEpoch,
		previousJustifiedEpoch: information.PreviousJustifiedEpoch,
		latestCrosslinks:       make([]primitives.Crosslink, len(information.LatestCrosslinks)),
		previousCrosslinks:     make([]primitives.Crosslink, len(information.PreviousCrosslinks)),
		earliestSlot:           information.Slot,
	}

	for i := range ei.slots {
		ei.slots[i] = make([]primitives.ShardAndCommittee, len(information.ShardCommitteesForSlots[i].Committees))
		for j := range ei.slots[i] {
			sc, err := primitives.ShardAndCommitteeFromProto(information.ShardCommitteesForSlots[i].Committees[j])
			if err != nil {
				return nil, err
			}

			ei.slots[i][j] = *sc
		}
	}

	for i := range ei.slotsNextShuffling {
		ei.slotsNextShuffling[i] = make([]primitives.ShardAndCommittee, len(information.ShardCommitteesForNextEpoch[i].Committees))
		for j := range ei.slotsNextShuffling[i] {
			sc, err := primitives.ShardAndCommitteeFromProto(information.ShardCommitteesForNextEpoch[i].Committees[j])
			if err != nil {
				return nil, err
			}

			ei.slotsNextShuffling[i][j] = *sc
		}
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

// SlotProposalAssignment tells a validator to propose on a certain shard.
type SlotProposalAssignment struct {
	Shard     uint64
	Validator uint32
}

// Manager is a manager that keeps track of multiple validators.
type Manager struct {
	blockchainRPC          pb.BlockchainRPCClient
	shardRPC               pb.ShardRPCClient
	validatorMap           map[uint32]*Validator
	keystore               Keystore
	latestEpochInformation epochInformation
	epochIndex             int64
	currentSlot            uint64
	config                 *config.Config
	synced                 bool
	genesisTime            uint64
	toPropose              map[uint64][]SlotProposalAssignment
}

// NewManager creates a new validator manager to manage some validators.
func NewManager(ctx context.Context, blockchainRPC pb.BlockchainRPCClient, shardRPC pb.ShardRPCClient, validators []uint32, keystore Keystore, c *config.Config) (*Manager, error) {
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
		v, err := NewValidator(ctx, keystore, blockchainRPC, shardRPC, validators[idx], c, forkData)
		if err != nil {
			return nil, err
		}
		validatorObjs[uint32(id)] = v
	}

	vm := &Manager{
		blockchainRPC: blockchainRPC,
		shardRPC:      shardRPC,
		toPropose:     make(map[uint64][]SlotProposalAssignment),
		validatorMap:  validatorObjs,
		keystore:      keystore,
		config:        c,
		currentSlot:   0,
		epochIndex:    -1,
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

	if !epochInformation.HasEpochInformation {
		return nil
	}

	ei, err := epochInformationFromProto(epochInformation.Information)
	if err != nil {
		return err
	}

	if vm.epochIndex != int64(slotNumber/vm.config.EpochLength) {
		// go through each committee in the current epoch
		for _, slotCommittees := range ei.slots[vm.config.EpochLength:] {
			for _, committee := range slotCommittees {
				for s := 0; uint64(s) < vm.config.EpochLength; s++ {
					// slot relative to earliest slot
					slot := int64(s) + ei.earliestSlot + 1 + int64(vm.config.EpochLength)

					proposer := committee.Committee[int(slot+1)%len(committee.Committee)]

					if _, found := vm.validatorMap[proposer]; found {
						vm.toPropose[uint64(slot)] = append(vm.toPropose[uint64(slot)], SlotProposalAssignment{
							Shard:     committee.Shard,
							Validator: proposer,
						})
					}
				}
			}

		}

		shardsToSubscribe := map[uint64]struct{}{}

		// find the committee pertaining to that shard
		for s := 0; uint64(s) < vm.config.EpochLength; s++ {
			// slot relative to earliest slot
			slot := int64(s) + ei.earliestSlot + int64(vm.config.EpochLength) + 1

			slotIndex := vm.config.EpochLength + uint64(s)
			for i := range ei.slots[slotIndex] {
				slotCommittee := ei.slots[slotIndex][i]

				propserNonshuffling := slotCommittee.Committee[int(slot)%len(slotCommittee.Committee)]

				slotNextCommittee := ei.slotsNextShuffling[s][i]
				proposerShuffling := slotNextCommittee.Committee[int(slot)%len(slotNextCommittee.Committee)]

				if _, found := vm.validatorMap[propserNonshuffling]; found {
					shardsToSubscribe[slotCommittee.Shard] = struct{}{}
				}

				if _, found := vm.validatorMap[proposerShuffling]; found {
					shardsToSubscribe[slotNextCommittee.Shard] = struct{}{}
				}
			}
		}

		for shardID := range shardsToSubscribe {
			_, err := vm.shardRPC.SubscribeToShard(context.Background(), &pb.ShardSubscribeRequest{
				ShardID:       shardID,
				CrosslinkSlot: ei.latestCrosslinks[shardID].Slot,
				BlockHash:     ei.latestCrosslinks[shardID].ShardBlockHash[:],
			})
			if err != nil {
				return err
			}
		}
	}

	vm.latestEpochInformation = *ei
	vm.synced = true
	vm.epochIndex = int64(slotNumber / vm.config.EpochLength)

	return nil
}

// NewSlot is run when a new slot starts.
func (vm *Manager) NewSlot(slotNumber uint64) error {
	earliestSlot := vm.latestEpochInformation.earliestSlot
	logrus.WithField("slot", slotNumber).Debug("heard new slot")

	proposerSlotCommittees := vm.latestEpochInformation.slots[int64(slotNumber-1)-earliestSlot]

	proposer := proposerSlotCommittees[0].Committee[(slotNumber-1)%uint64(len(proposerSlotCommittees[0].Committee))]
	if validator, found := vm.validatorMap[proposer]; found {
		err := validator.proposeBlock(context.Background(), proposerAssignment{
			slot: uint64(slotNumber),
		})
		if err != nil {
			return err
		}
	}

	assignments, found := vm.toPropose[slotNumber]
	if found {
		for _, assignment := range assignments {
			if validator, found := vm.validatorMap[assignment.Validator]; found {
				err := validator.proposeShardblock(context.Background(), assignment.Shard, slotNumber, chainhash.Hash{})
				if err != nil {
					return err
				}
			}
		}

		delete(vm.toPropose, slotNumber)
	}

	halfSlot := time.Unix(int64(slotNumber*uint64(vm.config.SlotDuration)+vm.genesisTime+uint64((vm.config.SlotDuration+1)/2)), 5e8)

	<-time.NewTimer(halfSlot.Sub(utils.Now())).C

	logrus.WithField("slot", slotNumber).Debug("requesting epoch information")
	if err := vm.UpdateEpochInformation(slotNumber); err != nil {
		return err
	}
	logrus.WithField("index", vm.epochIndex).Debug("got epoch information")

	earliestSlot = vm.latestEpochInformation.earliestSlot

	slotToAttest := slotNumber

	slotCommittees := vm.latestEpochInformation.slots[int64(slotToAttest)-earliestSlot-1] // we actually want to attest MinAttestationInclusionDistance after the slot

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
					targetEpoch := vm.epochIndex
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
						targetEpoch:      uint64(targetEpoch),
						targetHash:       targetHash,
					})
					if err != nil {
						return err
					}

					_, err = vm.blockchainRPC.SubmitAttestation(context.Background(), att.ToProto())
					if err != nil {
						return err
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

	if nextEpochSlot < currentSlot {
		nextEpochSlot = currentSlot
	}

	genesisTime := state.GenesisTime

	vm.genesisTime = genesisTime

	nextSlotTime := time.Unix(int64(nextEpochSlot*uint64(vm.config.SlotDuration)+genesisTime), 5e8)
	slotNumber := nextEpochSlot

	<-time.NewTimer(nextSlotTime.Sub(utils.Now())).C

	logrus.WithField("slot", slotNumber).Debug("requesting epoch information")
	if err := vm.UpdateEpochInformation(slotNumber); err != nil {
		return err
	}
	logrus.WithField("index", vm.epochIndex).Debug("got epoch information")

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
