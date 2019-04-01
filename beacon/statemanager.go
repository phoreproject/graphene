package beacon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/phoreproject/synapse/beacon/db"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"

	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
)

// StateManager handles all state transitions, storing of states for different forks,
// and time-based state updates.
type StateManager struct {
	config       *config.Config
	state        primitives.State // state of the head
	stateMap     map[chainhash.Hash]primitives.State
	db           db.Database
	blockchain   *Blockchain
	stateMapLock *sync.RWMutex
	stateLock    *sync.Mutex
	genesisTime  uint64
	currentEpoch uint64
}

// NewStateManager creates a new state manager.
func NewStateManager(c *config.Config, initialValidators []InitialValidatorEntry, genesisTime uint64, skipValidation bool, blockchain *Blockchain, db db.Database) (*StateManager, error) {
	s := &StateManager{
		config:       c,
		stateMap:     make(map[chainhash.Hash]primitives.State),
		stateMapLock: new(sync.RWMutex),
		stateLock:    new(sync.Mutex),
		genesisTime:  genesisTime,
		blockchain:   blockchain,
		db:           db,
	}
	intialState, err := InitializeState(c, initialValidators, genesisTime, skipValidation)
	if err != nil {
		return nil, err
	}
	s.state = *intialState
	return s, nil
}

// GetGenesisTime gets the time of the genesis slot.
func (sm *StateManager) GetGenesisTime() uint64 {
	return sm.genesisTime
}

// GetStateForHash gets the state for a certain block hash.
func (sm *StateManager) GetStateForHash(blockHash chainhash.Hash) (*primitives.State, bool) {
	sm.stateMapLock.RLock()
	state, found := sm.stateMap[blockHash]
	sm.stateMapLock.RUnlock()
	if !found {
		return nil, false
	}
	return &state, true
}

// UpdateHead updates the head state to a state in the stateMap.
func (sm *StateManager) UpdateHead(blockHash chainhash.Hash) error {
	sm.stateMapLock.RLock()
	state, found := sm.stateMap[blockHash]
	if !found {
		return fmt.Errorf("couldn't find block with hash %s in state map", blockHash)
	}
	sm.stateMapLock.RUnlock()
	sm.stateLock.Lock()
	logrus.WithField("slot", state.Slot).Debug("setting state head")
	sm.state = state
	sm.stateLock.Unlock()
	return nil
}

// GetHeadSlot gets the slot of the head state.
func (sm *StateManager) GetHeadSlot() uint64 {
	return sm.state.Slot
}

// GetHeadState gets the head state.
func (sm *StateManager) GetHeadState() primitives.State {
	return sm.state
}

// InitializeState initializes state to the genesis state according to the config.
func InitializeState(c *config.Config, initialValidators []InitialValidatorEntry, genesisTime uint64, skipValidation bool) (*primitives.State, error) {
	crosslinks := make([]primitives.Crosslink, c.ShardCount)

	for i := 0; i < c.ShardCount; i++ {
		crosslinks[i] = primitives.Crosslink{
			Slot:           c.InitialSlotNumber,
			ShardBlockHash: zeroHash,
		}
	}

	recentBlockHashes := make([]chainhash.Hash, c.LatestBlockRootsLength)
	for i := uint64(0); i < c.LatestBlockRootsLength; i++ {
		recentBlockHashes[i] = zeroHash
	}

	initialState := primitives.State{
		Slot:        0,
		GenesisTime: genesisTime,
		ForkData: primitives.ForkData{
			PreForkVersion:  c.InitialForkVersion,
			PostForkVersion: c.InitialForkVersion,
			ForkSlotNumber:  c.InitialSlotNumber,
		},
		ValidatorRegistry:                 []primitives.Validator{},
		ValidatorBalances:                 []uint64{},
		ValidatorRegistryLatestChangeSlot: c.InitialSlotNumber,
		ValidatorRegistryExitCount:        0,
		ValidatorRegistryDeltaChainTip:    chainhash.Hash{},

		RandaoMix:                 chainhash.Hash{},
		NextSeed:                  chainhash.Hash{},
		ShardAndCommitteeForSlots: [][]primitives.ShardAndCommittee{},

		PreviousJustifiedSlot: c.InitialSlotNumber,
		JustifiedSlot:         c.InitialSlotNumber,
		JustificationBitfield: 0,
		FinalizedSlot:         c.InitialSlotNumber,

		LatestCrosslinks:            crosslinks,
		LatestBlockHashes:           recentBlockHashes,
		LatestPenalizedExitBalances: []uint64{},
		LatestAttestations:          []primitives.PendingAttestation{},
		BatchedBlockRoots:           []chainhash.Hash{},
	}

	for _, deposit := range initialValidators {
		pub, err := bls.DeserializePublicKey(deposit.PubKey)
		if err != nil {
			return nil, err
		}
		validatorIndex, err := initialState.ProcessDeposit(pub, deposit.DepositSize, deposit.ProofOfPossession, deposit.WithdrawalCredentials, skipValidation, c)
		if err != nil {
			return nil, err
		}
		if initialState.GetEffectiveBalance(validatorIndex, c) == c.MaxDeposit {
			err := initialState.UpdateValidatorStatus(validatorIndex, primitives.Active, c)
			if err != nil {
				return nil, err
			}
		}
	}

	initialShuffling := GetNewShuffling(zeroHash, initialState.ValidatorRegistry, 0, c)
	initialState.ShardAndCommitteeForSlots = append(initialShuffling, initialShuffling...)

	return &initialState, nil
}

// applyAttestation verifies and applies an attestation to the given state.
func (sm *StateManager) applyAttestation(s *primitives.State, att primitives.Attestation, c *config.Config) error {
	if att.Data.Slot+c.MinAttestationInclusionDelay > s.Slot {
		return errors.New("attestation included too soon")
	}

	if att.Data.Slot+c.EpochLength < s.Slot {
		return errors.New("attestation was not included within 1 epoch")
	}

	expectedJustifiedSlot := s.JustifiedSlot
	if att.Data.Slot < s.Slot-(s.Slot%c.EpochLength) {
		expectedJustifiedSlot = s.PreviousJustifiedSlot
	}

	if att.Data.JustifiedSlot != expectedJustifiedSlot {
		return errors.New("justified slot did not match expected justified slot")
	}

	node, err := sm.blockchain.GetHashBySlot(att.Data.JustifiedSlot)
	if err != nil {
		return err
	}

	if !att.Data.JustifiedBlockHash.IsEqual(&node) {
		return errors.New("justified block hash did not match")
	}

	if len(s.LatestCrosslinks) <= int(att.Data.Shard) {
		return errors.New("invalid shard number")
	}

	latestCrosslinkRoot := s.LatestCrosslinks[att.Data.Shard].ShardBlockHash

	if !att.Data.LatestCrosslinkHash.IsEqual(&latestCrosslinkRoot) && !att.Data.ShardBlockHash.IsEqual(&latestCrosslinkRoot) {
		return errors.New("latest crosslink is invalid")
	}

	participants, err := s.GetAttestationParticipants(att.Data, att.ParticipationBitfield, c)
	if err != nil {
		return err
	}

	dataRoot, err := ssz.TreeHash(primitives.AttestationDataAndCustodyBit{Data: att.Data, PoCBit: false})
	if err != nil {
		return err
	}

	groupPublicKey := bls.NewAggregatePublicKey()
	for _, p := range participants {
		pub, err := s.ValidatorRegistry[p].GetPublicKey()
		if err != nil {
			return err
		}
		groupPublicKey.AggregatePubKey(pub)
	}

	aggSig, err := bls.DeserializeSignature(att.AggregateSig)
	if err != nil {
		return err
	}

	valid, err := bls.VerifySig(groupPublicKey, dataRoot[:], aggSig, primitives.GetDomain(s.ForkData, att.Data.Slot, bls.DomainAttestation))
	if err != nil {
		return err
	}

	if !valid {
		return errors.New("attestation signature is invalid")
	}

	node, err = sm.blockchain.GetHashBySlot(att.Data.Slot)
	if err != nil {
		return err
	}

	if !att.Data.BeaconBlockHash.IsEqual(&node) {
		return errors.New("beacon block hash is invalid")
	}

	// REMOVEME
	if !att.Data.ShardBlockHash.IsEqual(&zeroHash) {
		return errors.New("invalid block hash")
	}

	s.LatestAttestations = append(s.LatestAttestations, primitives.PendingAttestation{
		Data:                  att.Data,
		ParticipationBitfield: att.ParticipationBitfield,
		CustodyBitfield:       att.CustodyBitfield,
		SlotIncluded:          s.Slot,
	})

	return nil
}

// processBlock tries to apply a block to the state.
func (sm *StateManager) processBlock(block *primitives.Block, newState *primitives.State) error {
	proposerIndex, err := newState.GetBeaconProposerIndex(newState.Slot-1, block.BlockHeader.SlotNumber-1, sm.config)
	if err != nil {
		return err
	}

	if block.BlockHeader.SlotNumber != newState.Slot {
		return errors.New("block has incorrect slot number")
	}

	blockWithoutSignature := block.Copy()
	blockWithoutSignature.BlockHeader.Signature = bls.EmptySignature.Serialize()
	blockWithoutSignatureRoot, err := ssz.TreeHash(blockWithoutSignature)
	if err != nil {
		return err
	}

	proposal := primitives.ProposalSignedData{
		Slot:      newState.Slot,
		Shard:     sm.config.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot, err := ssz.TreeHash(proposal)
	if err != nil {
		return err
	}

	proposerPub, err := newState.ValidatorRegistry[proposerIndex].GetPublicKey()
	if err != nil {
		return err
	}

	proposerSig, err := bls.DeserializeSignature(block.BlockHeader.Signature)
	if err != nil {
		return err
	}

	valid, err := bls.VerifySig(proposerPub, proposalRoot[:], proposerSig, bls.DomainProposal)
	if err != nil {
		return err
	}

	if !valid {
		return fmt.Errorf("block had invalid signature (expected signature from validator %d)", proposerIndex)
	}

	proposer := &newState.ValidatorRegistry[proposerIndex]

	var proposerSlotsBytes [8]byte
	binary.BigEndian.PutUint64(proposerSlotsBytes[:], proposer.ProposerSlots)

	randaoSig, err := bls.DeserializeSignature(block.BlockHeader.RandaoReveal)
	if err != nil {
		return err
	}

	valid, err = bls.VerifySig(proposerPub, proposerSlotsBytes[:], randaoSig, bls.DomainRandao)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("block has invalid randao signature")
	}

	randaoRevealSerialized, err := ssz.TreeHash(block.BlockHeader.RandaoReveal)
	if err != nil {
		return err
	}

	for i := range newState.RandaoMix {
		newState.RandaoMix[i] ^= randaoRevealSerialized[i]
	}

	if len(block.BlockBody.ProposerSlashings) > sm.config.MaxProposerSlashings {
		return errors.New("more than maximum proposer slashings")
	}

	if len(block.BlockBody.CasperSlashings) > sm.config.MaxCasperSlashings {
		return errors.New("more than maximum casper slashings")
	}

	if len(block.BlockBody.Attestations) > sm.config.MaxAttestations {
		return errors.New("more than maximum attestations")
	}

	if len(block.BlockBody.Exits) > sm.config.MaxExits {
		return errors.New("more than maximum exits")
	}

	if len(block.BlockBody.Deposits) > sm.config.MaxDeposits {
		return errors.New("more than maximum deposits")
	}

	for _, s := range block.BlockBody.ProposerSlashings {
		err := newState.ApplyProposerSlashing(s, sm.config)
		if err != nil {
			return err
		}
	}

	for _, c := range block.BlockBody.CasperSlashings {
		err := newState.ApplyCasperSlashing(c, sm.config)
		if err != nil {
			return err
		}
	}

	for _, a := range block.BlockBody.Attestations {
		err := sm.applyAttestation(newState, a, sm.config)
		if err != nil {
			return err
		}
	}

	// process deposits here

	for _, e := range block.BlockBody.Exits {
		err := newState.ApplyExit(e, sm.config)
		if err != nil {
			return err
		}
	}

	// VERIFY BLOCK STATE ROOT MATCHES STATE ROOT FROM PREVIOUS BLOCK IF NEEDED
	return nil
}

func (sm *StateManager) processSlot(newState *primitives.State, lastBlockHash chainhash.Hash) error {
	// increase the slot number
	newState.Slot++

	newState.LatestBlockHashes[(newState.Slot-1)%sm.config.LatestBlockRootsLength] = lastBlockHash

	if newState.Slot%sm.config.LatestBlockRootsLength == 0 {
		latestBlockHashesRoot, err := ssz.TreeHash(newState.LatestBlockHashes)
		if err != nil {
			return err
		}
		newState.BatchedBlockRoots = append(newState.BatchedBlockRoots, latestBlockHashesRoot)
	}

	return nil
}

func (sm *StateManager) processEpochTransition(newState *primitives.State) error {
	logrus.Debug("epoch transition")

	activeValidatorIndices := primitives.GetActiveValidatorIndices(newState.ValidatorRegistry)
	totalBalance := newState.GetTotalBalance(activeValidatorIndices, sm.config)

	// currentEpochAttestations is any attestation that happened in the last epoch
	currentEpochAttestations := []primitives.PendingAttestation{}
	for _, a := range newState.LatestAttestations {
		if newState.Slot-sm.config.EpochLength <= a.Data.Slot && a.Data.Slot < newState.Slot {
			currentEpochAttestations = append(currentEpochAttestations, a)
		}
	}

	previousEpochBoundaryHash, err := sm.blockchain.GetHashBySlot(newState.Slot - sm.config.EpochLength)
	if err != nil {
		previousEpochBoundaryHash = chainhash.Hash{}
	}

	// currentEpochBoundaryAttestations is any attestation in the last epoch that has its boundary hash set to
	// the last epoch boundary
	currentEpochBoundaryAttestations := []primitives.PendingAttestation{}
	for _, a := range currentEpochAttestations {
		if a.Data.EpochBoundaryHash.IsEqual(&previousEpochBoundaryHash) && a.Data.JustifiedSlot == newState.JustifiedSlot {
			currentEpochBoundaryAttestations = append(currentEpochBoundaryAttestations, a)
		}
	}

	// currentEpochBoundaryAttesterIndices are all participant validator IDs of attestations
	// with the epoch boundary set to the last epoch boundary.
	currentEpochBoundaryAttesterIndices := map[uint32]struct{}{}
	for _, a := range currentEpochBoundaryAttestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, sm.config)
		if err != nil {
			return err
		}
		for _, p := range participants {
			currentEpochBoundaryAttesterIndices[p] = struct{}{}
		}
	}
	currentEpochBoundaryAttestingBalance := newState.GetTotalBalanceMap(currentEpochBoundaryAttesterIndices, sm.config)

	// previousEpochAttestations is any attestation in the last epoch
	previousEpochAttestations := []primitives.PendingAttestation{}
	for _, a := range newState.LatestAttestations {
		if newState.Slot-2*sm.config.EpochLength <= a.Data.Slot && a.Data.Slot < newState.Slot-sm.config.EpochLength {
			previousEpochAttestations = append(previousEpochAttestations, a)
		}
	}

	// previousEpochAttesterIndices are all participants of attestations in the previous epoch
	previousEpochAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochAttestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, sm.config)
		if err != nil {
			return err
		}
		for _, p := range participants {
			previousEpochAttesterIndices[p] = struct{}{}
		}
	}

	// previousEpochJustifiedAttestations are any attestations in the previous epoch that have a
	// justified slot equal to the previous justified slot.
	previousEpochJustifiedAttestations := []primitives.PendingAttestation{}
	for _, a := range previousEpochAttestations {
		if a.Data.JustifiedSlot == newState.PreviousJustifiedSlot {
			previousEpochJustifiedAttestations = append(previousEpochJustifiedAttestations, a)
		}
	}
	for _, a := range currentEpochAttestations {
		if a.Data.JustifiedSlot == newState.PreviousJustifiedSlot {
			previousEpochJustifiedAttestations = append(previousEpochJustifiedAttestations, a)
		}
	}

	// previousEpochJustifiedAttesterIndices are all participants of attestations in the previous
	// epoch with a justified slot equal to the previous justified slot.
	previousEpochJustifiedAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochJustifiedAttestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, sm.config)
		if err != nil {
			return err
		}
		for _, p := range participants {
			previousEpochJustifiedAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochJustifiedAttestingBalance := newState.GetTotalBalanceMap(previousEpochJustifiedAttesterIndices, sm.config)

	epochBoundaryHashMinus2 := chainhash.Hash{}
	if newState.Slot >= 2*sm.config.EpochLength {
		ebhm2, err := sm.blockchain.GetHashBySlot(newState.Slot - 2*sm.config.EpochLength)
		if err != nil {
			ebhm2 = chainhash.Hash{}
		}
		epochBoundaryHashMinus2 = ebhm2
	}

	// previousEpochBoundaryAttestations is any attestation in the previous epoch where the epoch boundary is
	// set to the epoch boundary two epochs ago
	previousEpochBoundaryAttestations := []primitives.PendingAttestation{}
	for _, a := range previousEpochJustifiedAttestations {
		if epochBoundaryHashMinus2.IsEqual(&a.Data.EpochBoundaryHash) {
			previousEpochBoundaryAttestations = append(previousEpochBoundaryAttestations, a)
		}
	}

	previousEpochBoundaryAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochBoundaryAttestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, sm.config)
		if err != nil {
			return err
		}
		for _, p := range participants {
			previousEpochBoundaryAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochBoundaryAttestingBalance := newState.GetTotalBalanceMap(previousEpochBoundaryAttesterIndices, sm.config)

	previousEpochHeadAttestations := []primitives.PendingAttestation{}
	for _, a := range previousEpochAttestations {
		blockRoot, err := sm.blockchain.GetHashBySlot(a.Data.Slot)
		if err != nil {
			break
		}
		if a.Data.BeaconBlockHash.IsEqual(&blockRoot) {
			previousEpochHeadAttestations = append(previousEpochHeadAttestations, a)
		}
	}

	previousEpochHeadAttesterIndices := map[uint32]struct{}{}
	for _, a := range previousEpochHeadAttestations {
		participants, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, sm.config)
		if err != nil {
			return err
		}
		for _, p := range participants {
			previousEpochHeadAttesterIndices[p] = struct{}{}
		}
	}

	previousEpochHeadAttestingBalance := newState.GetTotalBalanceMap(previousEpochHeadAttesterIndices, sm.config)

	newState.PreviousJustifiedSlot = newState.JustifiedSlot
	newState.JustificationBitfield = newState.JustificationBitfield * 2

	logger.WithFields(logger.Fields{
		"previousAttestingBalance": previousEpochBoundaryAttestingBalance,
		"currentAttestingBalance":  currentEpochBoundaryAttestingBalance,
		"totalBalance":             totalBalance,
	}).Debug("updating justified/finalized state")

	if 3*previousEpochBoundaryAttestingBalance >= 2*totalBalance {
		newState.JustificationBitfield |= 2 // mark the last epoch as justified
		newState.JustifiedSlot = newState.Slot - 2*sm.config.EpochLength
	}

	if 3*currentEpochBoundaryAttestingBalance >= 2*totalBalance {
		newState.JustificationBitfield |= 1 // mark the current epoch as justified
		newState.JustifiedSlot = newState.Slot - sm.config.EpochLength
	}

	// if 3 of the last 4, 7 of the last 8, or 14 of the last 16 blocks were justified, finalize the last justified slot
	if (newState.PreviousJustifiedSlot == newState.Slot-2*sm.config.EpochLength && newState.JustificationBitfield%4 == 3) ||
		(newState.PreviousJustifiedSlot == newState.Slot-3*sm.config.EpochLength && newState.JustificationBitfield%8 == 7) ||
		(newState.PreviousJustifiedSlot == newState.Slot-4*sm.config.EpochLength && newState.JustificationBitfield%16 > 14) {
		newState.FinalizedSlot = newState.PreviousJustifiedSlot
	}

	// attestingValidatorIndices gets the participants that attested to a certain shardblockRoot for a certain shardCommittee
	attestingValidatorIndices := func(shardComittee primitives.ShardAndCommittee, shardBlockRoot chainhash.Hash) ([]uint32, error) {
		outMap := map[uint32]struct{}{}
		for _, a := range currentEpochAttestations {
			if a.Data.Shard == shardComittee.Shard && a.Data.ShardBlockHash.IsEqual(&shardBlockRoot) {
				for i, s := range shardComittee.Committee {
					bit := a.ParticipationBitfield[i/8] & (1 << uint(i%8))
					if bit != 0 {
						outMap[s] = struct{}{}
					}
				}
			}
		}
		for _, a := range previousEpochAttestations {
			if a.Data.Shard == shardComittee.Shard && a.Data.ShardBlockHash.IsEqual(&shardBlockRoot) {
				for i, s := range shardComittee.Committee {
					bit := a.ParticipationBitfield[i/8] & (1 << uint(i%8))
					if bit != 0 {
						outMap[s] = struct{}{}
					}
				}
			}
		}
		out := make([]uint32, len(outMap))
		i := 0
		for id := range outMap {
			out[i] = id
			i++
		}
		return out, nil
	}

	// winningRoot finds the winning shard block hash
	winningRoot := func(shardCommittee primitives.ShardAndCommittee) (*chainhash.Hash, error) {
		balances := map[chainhash.Hash]struct{}{}
		for _, a := range currentEpochAttestations {
			if a.Data.Shard != shardCommittee.Shard {
				continue
			}
			balances[a.Data.ShardBlockHash] = struct{}{}
		}
		for _, a := range previousEpochAttestations {
			if a.Data.Shard != shardCommittee.Shard {
				continue
			}
			balances[a.Data.ShardBlockHash] = struct{}{}
		}

		topBalance := uint64(0)
		topHash := chainhash.Hash{1}

		for b := range balances {
			validatorIndices, err := attestingValidatorIndices(shardCommittee, b)
			if err != nil {
				return nil, err
			}

			sumBalance := newState.GetTotalBalance(validatorIndices, sm.config)

			if sumBalance > totalBalance {
				topHash = b
				topBalance = sumBalance
			}

			if sumBalance == totalBalance {
				if bytes.Compare(topHash[:], b[:]) > 0 {
					topHash = b
				}
			}
		}
		if topBalance == 0 {
			return nil, nil
		}
		return &topHash, nil
	}

	shardWinnerCache := make([]map[uint64]chainhash.Hash, len(newState.ShardAndCommitteeForSlots))

	for i, shardCommitteeAtSlot := range newState.ShardAndCommitteeForSlots {
		for _, shardCommittee := range shardCommitteeAtSlot {
			bestRoot, err := winningRoot(shardCommittee)
			if err != nil {
				return err
			}
			if bestRoot == nil {
				continue
			}
			if shardWinnerCache[i] == nil {
				shardWinnerCache[i] = make(map[uint64]chainhash.Hash)
			}
			shardWinnerCache[i][shardCommittee.Shard] = *bestRoot
			attestingCommittee, err := attestingValidatorIndices(shardCommittee, *bestRoot)
			if err != nil {
				return err
			}
			totalAttestingBalance := newState.GetTotalBalance(attestingCommittee, sm.config)
			totalBalance := newState.GetTotalBalance(shardCommittee.Committee, sm.config)

			if 3*totalAttestingBalance >= 2*totalBalance {
				newState.LatestCrosslinks[shardCommittee.Shard] = primitives.Crosslink{
					Slot:           newState.Slot,
					ShardBlockHash: *bestRoot,
				}
				logrus.WithFields(logrus.Fields{
					"slot":             newState.Slot,
					"shardBlockHash":   bestRoot.String(),
					"totalAttestation": totalAttestingBalance,
					"totalBalance":     totalBalance,
				}).Debug("crosslink created")
			}
		}
	}

	baseRewardQuotient := sm.config.BaseRewardQuotient * intSqrt(totalBalance*config.UnitInCoin)
	baseReward := func(index uint32) uint64 {
		return newState.GetEffectiveBalance(index, sm.config) / baseRewardQuotient / 5
	}

	inactivityPenalty := func(index uint32, epochsSinceFinality uint64) uint64 {
		return baseReward(index) + newState.GetEffectiveBalance(index, sm.config)*epochsSinceFinality/sm.config.InactivityPenaltyQuotient/2
	}

	epochsSinceFinality := (newState.Slot - newState.FinalizedSlot) / sm.config.EpochLength

	previousAttestationCache := map[uint32]*primitives.PendingAttestation{}
	for _, a := range previousEpochAttestations {
		participation, err := newState.GetAttestationParticipants(a.Data, a.ParticipationBitfield, sm.config)
		if err != nil {
			return err
		}

		for _, p := range participation {
			previousAttestationCache[p] = &a
		}
	}

	totalPenalized := uint64(0)
	totalRewarded := uint64(0)

	if epochsSinceFinality <= 4 {
		// any validator in previous_epoch_justified_attester_indices is rewarded
		for index := range previousEpochJustifiedAttesterIndices {
			totalRewarded += baseReward(index) * previousEpochJustifiedAttestingBalance / totalBalance
			newState.ValidatorBalances[index] += baseReward(index) * previousEpochJustifiedAttestingBalance / totalBalance
		}

		// any validator in previous_epoch_boundary_attester_indices is rewarded
		for index := range previousEpochBoundaryAttesterIndices {
			totalRewarded += baseReward(index) * previousEpochBoundaryAttestingBalance / totalBalance
			newState.ValidatorBalances[index] += baseReward(index) * previousEpochBoundaryAttestingBalance / totalBalance
		}

		// any validator in previous_epoch_head_attester_indices is rewarded
		for index := range previousEpochHeadAttesterIndices {
			totalRewarded += baseReward(index) * previousEpochHeadAttestingBalance / totalBalance
			newState.ValidatorBalances[index] += baseReward(index) * previousEpochHeadAttestingBalance / totalBalance
		}

		// any validator in previous_epoch_head_attester_indices is rewarded
		for index := range previousEpochAttesterIndices {
			inclusionDistance := previousAttestationCache[index].SlotIncluded - previousAttestationCache[index].Data.Slot
			totalRewarded += baseReward(index) * sm.config.MinAttestationInclusionDelay / inclusionDistance
			newState.ValidatorBalances[index] += baseReward(index) * sm.config.MinAttestationInclusionDelay / inclusionDistance
		}

		// any validator not in previous_epoch_head_attester_indices is slashed
		// any validator not in previous_epoch_boundary_attester_indices is slashed
		// any validator not in previous_epoch_justified_attester_indices is slashed

		if newState.Slot >= 2*sm.config.EpochLength {
			for idx, validator := range newState.ValidatorRegistry {
				index := uint32(idx)
				if validator.Status != primitives.Active {
					continue
				}
				if _, found := previousEpochHeadAttesterIndices[index]; !found {
					totalPenalized += baseReward(index)
					newState.ValidatorBalances[index] -= baseReward(index)
				}
				if _, found := previousEpochBoundaryAttesterIndices[index]; !found {
					totalPenalized += baseReward(index)
					newState.ValidatorBalances[index] -= baseReward(index)
				}
				if _, found := previousEpochJustifiedAttesterIndices[index]; !found {
					totalPenalized += baseReward(index)
					newState.ValidatorBalances[index] -= baseReward(index)
				}
			}
		}
	} else {
		// any validator not in previous_epoch_head_attester_indices is slashed
		for idx, validator := range newState.ValidatorRegistry {
			index := uint32(idx)
			if validator.Status == primitives.Active {
				if _, found := previousEpochJustifiedAttesterIndices[index]; !found {
					totalPenalized += inactivityPenalty(index, epochsSinceFinality)
					newState.ValidatorBalances[index] -= inactivityPenalty(index, epochsSinceFinality)
				}
				if _, found := previousEpochBoundaryAttesterIndices[index]; !found {
					totalPenalized += inactivityPenalty(index, epochsSinceFinality)
					newState.ValidatorBalances[index] -= inactivityPenalty(index, epochsSinceFinality)
				}
				if _, found := previousEpochHeadAttesterIndices[index]; !found {
					totalPenalized += baseReward(index)
					newState.ValidatorBalances[index] -= baseReward(index)
				}
			} else if validator.Status == primitives.ExitedWithPenalty {
				totalPenalized += inactivityPenalty(index, epochsSinceFinality) + baseReward(index)
				newState.ValidatorBalances[index] -= inactivityPenalty(index, epochsSinceFinality) + baseReward(index)
			}
		}
		for index := range previousEpochAttesterIndices {
			inclusionDistance := previousAttestationCache[index].SlotIncluded - previousAttestationCache[index].Data.Slot
			totalPenalized += baseReward(index) - baseReward(index)*sm.config.MinAttestationInclusionDelay/inclusionDistance
			newState.ValidatorBalances[index] -= baseReward(index) - baseReward(index)*sm.config.MinAttestationInclusionDelay/inclusionDistance
		}
	}

	for index := range previousEpochAttesterIndices {
		proposerIndex, err := newState.GetBeaconProposerIndex(newState.Slot-1, previousAttestationCache[index].SlotIncluded-1, sm.config)
		if err != nil {
			return err
		}
		totalRewarded += baseReward(index) / sm.config.IncluderRewardQuotient
		newState.ValidatorBalances[proposerIndex] += baseReward(index) / sm.config.IncluderRewardQuotient
	}

	if newState.Slot >= 2*sm.config.EpochLength {
		for slot, shardCommitteeAtSlot := range newState.ShardAndCommitteeForSlots[:sm.config.EpochLength] {
			for _, shardCommittee := range shardCommitteeAtSlot {
				winningRoot := shardWinnerCache[slot][shardCommittee.Shard]
				participationIndices, err := attestingValidatorIndices(shardCommittee, winningRoot)
				if err != nil {
					return err
				}

				participationIndicesMap := map[uint32]struct{}{}
				for _, p := range participationIndices {
					participationIndicesMap[p] = struct{}{}
				}

				totalAttestingBalance := newState.GetTotalBalance(participationIndices, sm.config)
				totalBalance := newState.GetTotalBalance(shardCommittee.Committee, sm.config)

				for _, index := range shardCommittee.Committee {
					if _, found := participationIndicesMap[index]; found {
						totalRewarded += baseReward(index) * totalAttestingBalance / totalBalance
						newState.ValidatorBalances[index] += baseReward(index) * totalAttestingBalance / totalBalance
					} else {
						totalPenalized += baseReward(index)
						newState.ValidatorBalances[index] -= baseReward(index)
					}
				}
			}
		}
	}

	for index, validator := range newState.ValidatorRegistry {
		if validator.Status == primitives.Active && newState.ValidatorBalances[index] < sm.config.EjectionBalance {
			err := newState.UpdateValidatorStatus(uint32(index), primitives.ExitedWithoutPenalty, sm.config)
			if err != nil {
				return err
			}
		}
	}
	logrus.WithField("totalRewarded", totalRewarded).WithField("totalPenalized", totalPenalized).WithField("netRewards", int64(totalRewarded)-int64(totalPenalized)).Debug("finished processing rewards")

	shouldUpdateRegistry := true

	if newState.FinalizedSlot <= newState.ValidatorRegistryLatestChangeSlot {
		shouldUpdateRegistry = false
	}

	for _, shardAndCommittees := range newState.ShardAndCommitteeForSlots {
		for _, committee := range shardAndCommittees {
			if newState.LatestCrosslinks[committee.Shard].Slot <= newState.ValidatorRegistryLatestChangeSlot {
				shouldUpdateRegistry = false
				goto done
			}
		}
	}
done:

	if shouldUpdateRegistry {
		err := newState.UpdateValidatorRegistry(sm.config)
		if err != nil {
			return err
		}

		newState.ValidatorRegistryLatestChangeSlot = newState.Slot
		copy(newState.ShardAndCommitteeForSlots[:sm.config.EpochLength], newState.ShardAndCommitteeForSlots[sm.config.EpochLength:])
		lastSlot := newState.ShardAndCommitteeForSlots[len(newState.ShardAndCommitteeForSlots)-1]
		lastCommittee := lastSlot[len(lastSlot)-1]
		nextStartShard := (lastCommittee.Shard + 1) % uint64(sm.config.ShardCount)
		newShuffling := GetNewShuffling(newState.RandaoMix, newState.ValidatorRegistry, int(nextStartShard), sm.config)
		copy(newState.ShardAndCommitteeForSlots[sm.config.EpochLength:], newShuffling)
	} else {
		copy(newState.ShardAndCommitteeForSlots[:sm.config.EpochLength], newState.ShardAndCommitteeForSlots[sm.config.EpochLength:])
		epochsSinceLastRegistryChange := (newState.Slot - newState.ValidatorRegistryLatestChangeSlot) / sm.config.EpochLength
		startShard := newState.ShardAndCommitteeForSlots[0][0].Shard

		// epochsSinceLastRegistryChange is a power of 2
		if epochsSinceLastRegistryChange&(epochsSinceLastRegistryChange-1) == 0 {
			newShuffling := GetNewShuffling(newState.RandaoMix, newState.ValidatorRegistry, int(startShard), sm.config)
			copy(newState.ShardAndCommitteeForSlots[sm.config.EpochLength:], newShuffling)
		}
	}

	newLatestAttestations := make([]primitives.PendingAttestation, 0)
	for _, a := range newState.LatestAttestations {
		if a.Data.Slot >= newState.Slot-2*sm.config.EpochLength {
			newLatestAttestations = append(newLatestAttestations, a)
		}
	}

	newState.LatestAttestations = newLatestAttestations

	return nil
}

// ProcessSlots uses the current head to process slots up to a certain slot, applying
// slot transitions and epoch transitions, and returns the updated state. Note that this should
// only process up to the current slot number so that the lastBlockHash remains constant.
func (sm *StateManager) ProcessSlots(upTo uint64, lastBlockHash chainhash.Hash) (*primitives.State, error) {
	sm.stateLock.Lock()
	defer sm.stateLock.Unlock()
	newState := sm.state.Copy()

	for newState.Slot < upTo {
		// this only happens when there wasn't a block at the first slot of the epoch
		if newState.Slot > 1 && sm.currentEpoch < newState.Slot/sm.config.EpochLength && newState.Slot%sm.config.EpochLength == 0 {
			logrus.Info("processing epoch transition")
			t := time.Now()
			err := sm.processEpochTransition(&newState)
			if err != nil {
				return nil, err
			}
			logrus.WithField("time", time.Since(t)).Debug("done processing epoch transition")
			sm.currentEpoch = newState.Slot / sm.config.EpochLength
		}

		err := sm.processSlot(&newState, lastBlockHash)
		if err != nil {
			return nil, err
		}

		logrus.WithFields(logrus.Fields{
			"slot":          newState.Slot,
			"lastBlockHash": lastBlockHash,
		}).Info("processing slot")
	}

	return &newState, nil
}

// SetBlockState sets the state for a certain block. This SHOULD ONLY
// BE USED FOR THE GENESIS BLOCK!
func (sm *StateManager) SetBlockState(blockHash chainhash.Hash, state *primitives.State) error {
	// add the state to the statemap
	sm.stateMapLock.Lock()
	defer sm.stateMapLock.Unlock()
	logrus.WithField("hash", blockHash.String()).Debug("setting block state")
	sm.stateMap[blockHash] = *state
	return nil
}

// AddBlockToStateMap processes the block and adds it to the state map.
func (sm *StateManager) AddBlockToStateMap(block *primitives.Block) (*primitives.State, error) {
	// this should have already been done, but just in case, we should make sure the state is
	// updated to the block's slot number.
	newState, err := sm.ProcessSlots(block.BlockHeader.SlotNumber, block.BlockHeader.ParentRoot)
	if err != nil {
		return nil, err
	}

	err = sm.processBlock(block, newState)
	if err != nil {
		return nil, err
	}

	// we do this with the lock so we don't process the epoch transition twice
	sm.stateLock.Lock()
	if newState.Slot%sm.config.EpochLength == 0 {
		err := sm.processEpochTransition(newState)
		if err != nil {
			return nil, err
		}

		sm.currentEpoch = newState.Slot / sm.config.EpochLength
	}
	sm.stateLock.Unlock()

	blockHash, err := ssz.TreeHash(block)
	if err != nil {
		return nil, err
	}

	err = sm.db.SetBlockState(blockHash, *newState)
	if err != nil {
		return nil, err
	}

	err = sm.SetBlockState(blockHash, newState)
	if err != nil {
		return nil, err
	}

	return newState, nil
}

// DeleteStateBeforeFinalizedSlot deletes any states before the current finalized slot.
func (sm *StateManager) DeleteStateBeforeFinalizedSlot(finalizedSlot uint64) error {
	sm.stateMapLock.Lock()
	defer sm.stateMapLock.Unlock()
	// once we finalize the block, we can get rid of any states before finalizedSlot
	for i := range sm.stateMap {
		// if it happened before finalized slot, we don't need it
		if sm.stateMap[i].Slot < finalizedSlot {
			delete(sm.stateMap, i)
			err := sm.db.DeleteStateForBlock(i)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
