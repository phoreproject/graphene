package primitives_test

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/validator"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/chainhash"

	"github.com/go-test/deep"
	"github.com/phoreproject/synapse/primitives"
)

func TestForkData_Copy(t *testing.T) {
	baseForkData := &primitives.ForkData{
		PreForkVersion:  0,
		PostForkVersion: 0,
		ForkSlotNumber:  0,
	}

	copyForkData := baseForkData.Copy()

	copyForkData.PreForkVersion = 1
	if baseForkData.PreForkVersion == 1 {
		t.Fatal("mutating preForkVersion mutates base")
	}

	copyForkData.PostForkVersion = 1
	if baseForkData.PostForkVersion == 1 {
		t.Fatal("mutating postForkVersion mutates base")
	}

	copyForkData.ForkSlotNumber = 1
	if baseForkData.ForkSlotNumber == 1 {
		t.Fatal("mutating forkSlotNumber mutates base")
	}
}

func TestForkData_ToFromProto(t *testing.T) {
	baseForkData := &primitives.ForkData{
		PreForkVersion:  1,
		PostForkVersion: 1,
		ForkSlotNumber:  1,
	}

	forkDataProto := baseForkData.ToProto()
	fromProto, err := primitives.ForkDataFromProto(forkDataProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseForkData); diff != nil {
		t.Fatal(diff)
	}
}

func TestState_Copy(t *testing.T) {
	baseState := &primitives.State{
		Slot:                               0,
		GenesisTime:                        0,
		ForkData:                           primitives.ForkData{},
		ValidatorRegistry:                  []primitives.Validator{},
		ValidatorBalances:                  []uint64{},
		ValidatorRegistryLatestChangeEpoch: 0,
		ValidatorRegistryExitCount:         0,
		ValidatorRegistryDeltaChainTip:     chainhash.Hash{},
		RandaoMix:                          chainhash.Hash{},
		ShardAndCommitteeForSlots:          [][]primitives.ShardAndCommittee{},
		PreviousJustifiedEpoch:             0,
		JustifiedEpoch:                     0,
		JustificationBitfield:              0,
		FinalizedEpoch:                     0,
		LatestCrosslinks:                   []primitives.Crosslink{},
		PreviousCrosslinks:                 []primitives.Crosslink{},
		LatestBlockHashes:                  []chainhash.Hash{},
		CurrentEpochAttestations:           []primitives.PendingAttestation{},
		PreviousEpochAttestations:          []primitives.PendingAttestation{},
		BatchedBlockRoots:                  []chainhash.Hash{},
		ShardRegistry:                      []chainhash.Hash{{}},
		Proposals:                          []primitives.ActiveProposal{},
		PendingVotes:                       []primitives.AggregatedVote{},
	}

	copyState := baseState.Copy()

	copyState.Slot = 1
	if baseState.Slot == 1 {
		t.Fatal("mutating slot mutates base")
	}

	copyState.GenesisTime = 1
	if baseState.GenesisTime == 1 {
		t.Fatal("mutating genesis time mutates base")
	}

	copyState.ForkData.ForkSlotNumber = 1
	if baseState.ForkData.ForkSlotNumber == 1 {
		t.Fatal("mutating fork data mutates base")
	}

	copyState.ValidatorRegistry = []primitives.Validator{
		{
			Status: 1,
		},
	}
	if len(baseState.ValidatorRegistry) != 0 {
		t.Fatal("mutating validator registry mutates base")
	}

	copyState.ValidatorBalances = []uint64{1}
	if len(baseState.ValidatorBalances) != 0 {
		t.Fatal("mutating validator balances mutates base")
	}

	copyState.ValidatorRegistryLatestChangeEpoch = 1
	if baseState.ValidatorRegistryLatestChangeEpoch == 1 {
		t.Fatal("mutating ValidatorRegistryLatestChangeSlot mutates base")
	}

	copyState.ValidatorRegistryExitCount = 1
	if baseState.ValidatorRegistryExitCount == 1 {
		t.Fatal("mutating ValidatorRegistryExitCount mutates base")
	}

	copyState.ValidatorRegistryDeltaChainTip[0] = 1
	if baseState.ValidatorRegistryDeltaChainTip[0] == 1 {
		t.Fatal("mutating ValidatorRegistryDeltaChainTip mutates base")
	}

	copyState.RandaoMix[0] = 1
	if baseState.RandaoMix[0] == 1 {
		t.Fatal("mutating RandaoMix mutates base")
	}

	copyState.ShardAndCommitteeForSlots = [][]primitives.ShardAndCommittee{{}}
	if len(baseState.ShardAndCommitteeForSlots) == 1 {
		t.Fatal("mutating ShardAndCommitteeForSlots mutates base")
	}

	copyState.JustifiedEpoch = 1
	if baseState.JustifiedEpoch == 1 {
		t.Fatal("mutating justifiedSlot mutates base")
	}

	copyState.JustificationBitfield = 1
	if baseState.JustificationBitfield == 1 {
		t.Fatal("mutating baseSlot mutates base")
	}

	copyState.FinalizedEpoch = 1
	if baseState.FinalizedEpoch == 1 {
		t.Fatal("mutating finalizedSlot mutates base")
	}

	copyState.LatestCrosslinks = []primitives.Crosslink{{}}
	if len(baseState.LatestCrosslinks) == 1 {
		t.Fatal("mutating latestCrosslinks mutates base")
	}

	copyState.PreviousCrosslinks = []primitives.Crosslink{{}}
	if len(baseState.PreviousCrosslinks) == 1 {
		t.Fatal("mutating latestCrosslinks mutates base")
	}

	copyState.LatestBlockHashes = []chainhash.Hash{{}}
	if len(baseState.LatestBlockHashes) == 1 {
		t.Fatal("mutating latestBlockHashes mutates base")
	}

	copyState.CurrentEpochAttestations = []primitives.PendingAttestation{{}}
	if len(baseState.CurrentEpochAttestations) == 1 {
		t.Fatal("mutating latestAttestations mutates base")
	}

	copyState.PreviousEpochAttestations = []primitives.PendingAttestation{{}}
	if len(baseState.PreviousEpochAttestations) == 1 {
		t.Fatal("mutating latestAttestations mutates base")
	}

	copyState.BatchedBlockRoots = []chainhash.Hash{{}}
	if len(baseState.BatchedBlockRoots) == 1 {
		t.Fatal("mutating batchedBlockRoots mutates base")
	}

	copyState.ShardRegistry = nil
	if baseState.ShardRegistry == nil {
		t.Fatal("mutating shardRegistry mutates base")
	}

	copyState.Proposals = []primitives.ActiveProposal{{}}
	if len(baseState.Proposals) != 0 {
		t.Fatal("mutating proposals mutates base")
	}

	copyState.PendingVotes = []primitives.AggregatedVote{{}}
	if len(baseState.PendingVotes) != 0 {
		t.Fatal("mutating proposals mutates base")
	}
}

func TestState_ToFromProto(t *testing.T) {
	baseState := &primitives.State{
		Slot:        1,
		GenesisTime: 1,
		ForkData:    primitives.ForkData{PreForkVersion: 1},
		ValidatorRegistry: []primitives.Validator{
			{
				Status: 1,
			},
		},
		ValidatorBalances:                  []uint64{1},
		ValidatorRegistryLatestChangeEpoch: 1,
		ValidatorRegistryExitCount:         1,
		ValidatorRegistryDeltaChainTip:     chainhash.Hash{1},
		RandaoMix:                          chainhash.Hash{1},
		ShardAndCommitteeForSlots:          [][]primitives.ShardAndCommittee{{{Shard: 1, Committee: []uint32{1}}}},
		PreviousJustifiedEpoch:             1,
		JustifiedEpoch:                     1,
		JustificationBitfield:              1,
		FinalizedEpoch:                     1,
		LatestCrosslinks:                   []primitives.Crosslink{{Slot: 1}},
		PreviousCrosslinks:                 []primitives.Crosslink{{Slot: 1}},
		LatestBlockHashes:                  []chainhash.Hash{{1}},
		CurrentEpochAttestations:           []primitives.PendingAttestation{{InclusionDelay: 1}},
		PreviousEpochAttestations:          []primitives.PendingAttestation{{InclusionDelay: 1}},
		BatchedBlockRoots:                  []chainhash.Hash{{1}},
		ShardRegistry:                      []chainhash.Hash{{1}},
		Proposals:                          []primitives.ActiveProposal{{StartEpoch: 1}},
		PendingVotes:                       []primitives.AggregatedVote{{Signature: [48]byte{1}}},
	}

	stateProto := baseState.ToProto()
	fromProto, err := primitives.StateFromProto(stateProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseState); diff != nil {
		t.Fatal(diff)
	}
}

func TestValidator_Copy(t *testing.T) {
	baseValidator := &primitives.Validator{
		Pubkey:                  [96]byte{},
		WithdrawalCredentials:   chainhash.Hash{},
		Status:                  0,
		LatestStatusChangeSlot:  0,
		ExitCount:               0,
		LastPoCChangeSlot:       0,
		SecondLastPoCChangeSlot: 0,
	}

	copyValidator := baseValidator.Copy()

	copyValidator.Pubkey[0] = 1
	if baseValidator.Pubkey[0] == 1 {
		t.Fatal("mutating pubkey mutates base")
	}

	copyValidator.WithdrawalCredentials[0] = 1
	if baseValidator.WithdrawalCredentials[0] == 1 {
		t.Fatal("mutating withdrawalCredentials mutates base")
	}

	copyValidator.Status = 1
	if baseValidator.Status == 1 {
		t.Fatal("mutating status mutates base")
	}

	copyValidator.LatestStatusChangeSlot = 1
	if baseValidator.LatestStatusChangeSlot == 1 {
		t.Fatal("mutating LatestStatusChangeSlot mutates base")
	}

	copyValidator.ExitCount = 1
	if baseValidator.ExitCount == 1 {
		t.Fatal("mutating ExitCount mutates base")
	}

	copyValidator.LastPoCChangeSlot = 1
	if baseValidator.LastPoCChangeSlot == 1 {
		t.Fatal("mutating LastPoCChangeSlot mutates base")
	}

	copyValidator.SecondLastPoCChangeSlot = 1
	if baseValidator.SecondLastPoCChangeSlot == 1 {
		t.Fatal("mutating SecondLastPoCChangeSlot mutates base")
	}
}

func TestValidator_ToFromProto(t *testing.T) {
	baseValidator := &primitives.Validator{
		Pubkey:                  [96]byte{1},
		WithdrawalCredentials:   chainhash.Hash{},
		Status:                  0,
		LatestStatusChangeSlot:  0,
		ExitCount:               0,
		LastPoCChangeSlot:       0,
		SecondLastPoCChangeSlot: 0,
	}

	validatorProto := baseValidator.ToProto()
	fromProto, err := primitives.ValidatorFromProto(validatorProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseValidator); diff != nil {
		t.Fatal(diff)
	}
}

func TestCrosslink_ToFromProto(t *testing.T) {
	baseCrosslink := &primitives.Crosslink{
		Slot:           1,
		ShardBlockHash: chainhash.Hash{1},
	}

	crosslinkProto := baseCrosslink.ToProto()
	fromProto, err := primitives.CrosslinkFromProto(crosslinkProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseCrosslink); diff != nil {
		t.Fatal(diff)
	}
}

func TestShardAndCommittee_Copy(t *testing.T) {
	baseShardCommitteee := &primitives.ShardAndCommittee{
		Shard:               0,
		Committee:           []uint32{},
		TotalValidatorCount: 0,
	}

	copyShardCommitee := baseShardCommitteee.Copy()

	copyShardCommitee.Shard = 1
	if baseShardCommitteee.Shard == 1 {
		t.Fatal("mutating shard mutates base")
	}

	copyShardCommitee.Committee = []uint32{1}
	if len(baseShardCommitteee.Committee) == 1 {
		t.Fatal("mutating committee mutates base")
	}

	copyShardCommitee.TotalValidatorCount = 1
	if baseShardCommitteee.TotalValidatorCount == 1 {
		t.Fatal("mutating TotalValidatorCount mutates base")
	}
}

func TestShardAndCommittee_ToFromProto(t *testing.T) {
	baseShardCommittee := &primitives.ShardAndCommittee{
		Shard:               1,
		Committee:           []uint32{1},
		TotalValidatorCount: 1,
	}

	shardCommitteeProto := baseShardCommittee.ToProto()
	fromProto, err := primitives.ShardAndCommitteeFromProto(shardCommitteeProto)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(fromProto, baseShardCommittee); diff != nil {
		t.Fatal(diff)
	}
}

// SetupState initializes state with a certain number of initial validators
func SetupState(initialValidators int, c *config.Config) (*primitives.State, validator.Keystore, error) {
	keystore := validator.NewFakeKeyStore()

	var validators []primitives.InitialValidatorEntry

	for i := 0; i <= initialValidators; i++ {
		key := keystore.GetKeyForValidator(uint32(i))
		pub := key.DerivePublicKey()
		hashPub, err := ssz.TreeHash(pub.Serialize())
		if err != nil {
			return nil, nil, err
		}
		proofOfPossession, err := bls.Sign(key, hashPub[:], bls.DomainDeposit)
		if err != nil {
			return nil, nil, err
		}
		validators = append(validators, primitives.InitialValidatorEntry{
			PubKey:                pub.Serialize(),
			ProofOfPossession:     proofOfPossession.Serialize(),
			WithdrawalShard:       1,
			WithdrawalCredentials: chainhash.Hash{},
			DepositSize:           c.MaxDeposit,
		})
	}

	s, err := primitives.InitializeState(c, validators, uint64(time.Now().Unix()), false)
	return s, &keystore, err
}

// FakeBlockView always returns 0-hash for all slots and the last state root.
type FakeBlockView struct{}

func (f FakeBlockView) GetHashBySlot(slot uint64) (chainhash.Hash, error) {
	return chainhash.Hash{}, nil
}

func (f FakeBlockView) Tip() (chainhash.Hash, error) {
	return chainhash.Hash{}, nil
}

func (f FakeBlockView) SetTipSlot(slot uint64) {}

func (f FakeBlockView) GetLastStateRoot() (chainhash.Hash, error) {
	return chainhash.Hash{}, nil
}

func SignBlock(block *primitives.Block, key *bls.SecretKey, c *config.Config) error {
	var slotBytes [8]byte
	binary.BigEndian.PutUint64(slotBytes[:], block.BlockHeader.SlotNumber)
	slotBytesHash := chainhash.HashH(slotBytes[:])

	slotBytesSignature, err := bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		return err
	}

	block.BlockHeader.RandaoReveal = slotBytesSignature.Serialize()
	block.BlockHeader.Signature = bls.EmptySignature.Serialize()

	blockWithoutSignatureRoot, err := ssz.TreeHash(block)
	if err != nil {
		return err
	}

	proposal := primitives.ProposalSignedData{
		Slot:      block.BlockHeader.SlotNumber,
		Shard:     c.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot, err := ssz.TreeHash(proposal)
	if err != nil {
		return err
	}

	blockSignature, err := bls.Sign(key, proposalRoot[:], bls.DomainProposal)
	if err != nil {
		return err
	}

	block.BlockHeader.Signature = blockSignature.Serialize()

	return nil
}

func TestProposerIndex(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	slotToPropose := state.Slot + 1

	stateSlot := state.EpochIndex * c.EpochLength
	slotIndex := int64(slotToPropose-1) - int64(stateSlot) + int64(stateSlot%c.EpochLength) + int64(c.EpochLength)

	firstCommittee := state.ShardAndCommitteeForSlots[slotIndex][0].Committee

	proposerIndex := firstCommittee[int(slotToPropose-1)%len(firstCommittee)]

	blockTest := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotToPropose,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: [48]byte{},
			Signature:    [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessBlock(blockTest, c, FakeBlockView{}, true)
	if err != nil {
		t.Fatal(err)
	}

	slotToPropose = state.Slot + 1

	stateSlot = state.EpochIndex * c.EpochLength
	slotIndex = int64(slotToPropose-1) - int64(stateSlot) + int64(stateSlot%c.EpochLength) + int64(c.EpochLength)

	firstCommittee = state.ShardAndCommitteeForSlots[slotIndex][0].Committee

	proposerIndex = firstCommittee[int(slotToPropose-1)%len(firstCommittee)]

	blockTest = &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotToPropose,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: [48]byte{},
			Signature:    [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex-1), c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	err = state.ProcessBlock(blockTest, c, FakeBlockView{}, true)
	if err == nil {
		t.Fatal("expected block with wrong validator to fail")
	}
}

func TestBlockMaximums(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	slotToPropose := state.Slot + 1

	stateSlot := state.EpochIndex * c.EpochLength
	slotIndex := int64(slotToPropose-1) - int64(stateSlot) + int64(stateSlot%c.EpochLength) + int64(c.EpochLength)

	firstCommittee := state.ShardAndCommitteeForSlots[slotIndex][0].Committee

	proposerIndex := firstCommittee[int(slotToPropose-1)%len(firstCommittee)]

	type BodyShouldValidate struct {
		b         primitives.BlockBody
		validates bool
		reason    string
	}

	// the regtest config allows 1 of each transaction
	testBlockBodies := []BodyShouldValidate{
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			true,
			"should validate with maximum number for all",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}, {}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			false,
			"should not validate with too many attestations",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}, {}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			false,
			"should not validate with too many deposits",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}, {}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			false,
			"should not validate with too many exits",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}, {}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			false,
			"should not validate with too many proposer slashings",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}, {}},
				Votes:             []primitives.AggregatedVote{{}},
			},
			false,
			"should not validate with too many CASPER slashings",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{{}},
				Deposits:          []primitives.Deposit{{}},
				Exits:             []primitives.Exit{{}},
				ProposerSlashings: []primitives.ProposerSlashing{{}},
				CasperSlashings:   []primitives.CasperSlashing{{}},
				Votes:             []primitives.AggregatedVote{{}, {}},
			},
			false,
			"should not validate with too many votes slashings",
		},
		{
			primitives.BlockBody{
				Attestations:      []primitives.Attestation{},
				Deposits:          []primitives.Deposit{},
				Exits:             []primitives.Exit{},
				ProposerSlashings: []primitives.ProposerSlashing{},
				CasperSlashings:   []primitives.CasperSlashing{},
				Votes:             []primitives.AggregatedVote{},
			},
			true,
			"should validate with empty block",
		},
	}

	for _, b := range testBlockBodies {
		blockTest := &primitives.Block{
			BlockHeader: primitives.BlockHeader{
				SlotNumber:   slotToPropose,
				ParentRoot:   chainhash.Hash{},
				StateRoot:    chainhash.Hash{},
				RandaoReveal: [48]byte{},
				Signature:    [48]byte{},
			},
			BlockBody: b.b,
		}

		err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
		if err != nil {
			t.Fatal(err)
		}

		stateCopy := state.Copy()

		err = stateCopy.ProcessSlot(chainhash.Hash{}, c)
		if err != nil {
			t.Fatal(err)
		}

		failedBecauseOfMax := false

		err = stateCopy.ProcessBlock(blockTest, c, FakeBlockView{}, true)

		if err != nil {
			failedBecauseOfMax = strings.Contains(err.Error(), "maximum")
		}

		if failedBecauseOfMax == b.validates {
			if b.validates {
				t.Fatalf("expected block to validate, but got error: %s", err)
			} else {
				t.Fatalf("expected block to fail with reason: %s", b.reason)
			}
		}
	}
}

func TestProposerSlashingValidation(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	// sign two conflicting blocks at the same height
	slotToPropose := state.Slot + 1
	proposerIndex, err := state.GetBeaconProposerIndex(slotToPropose-1, c)
	if err != nil {
		t.Fatal(err)
	}

	type BodyShouldValidate struct {
		ps        primitives.ProposerSlashing
		validates bool
		reason    string
	}

	block1 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotToPropose,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: [48]byte{},
			Signature:    [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	key := keystore.GetKeyForValidator(proposerIndex)

	var slotBytes [8]byte
	binary.BigEndian.PutUint64(slotBytes[:], block1.BlockHeader.SlotNumber)
	slotBytesHash := chainhash.HashH(slotBytes[:])

	slotBytesSignature, err := bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		t.Fatal(err)
	}

	block1.BlockHeader.RandaoReveal = slotBytesSignature.Serialize()
	block1.BlockHeader.Signature = bls.EmptySignature.Serialize()

	blockWithoutSignatureRoot, err := ssz.TreeHash(block1)
	if err != nil {
		t.Fatal(err)
	}

	proposal1 := primitives.ProposalSignedData{
		Slot:      block1.BlockHeader.SlotNumber,
		Shard:     c.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot1, err := ssz.TreeHash(proposal1)
	if err != nil {
		t.Fatal(err)
	}

	blockSignature1, err := bls.Sign(key, proposalRoot1[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	block2 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotToPropose,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: [48]byte{},
			Signature:    [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations:      []primitives.Attestation{{}},
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	binary.BigEndian.PutUint64(slotBytes[:], block2.BlockHeader.SlotNumber)
	slotBytesHash = chainhash.HashH(slotBytes[:])

	slotBytesSignature, err = bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		t.Fatal(err)
	}

	block2.BlockHeader.RandaoReveal = slotBytesSignature.Serialize()
	block2.BlockHeader.Signature = bls.EmptySignature.Serialize()

	blockWithoutSignatureRoot, err = ssz.TreeHash(block2)
	if err != nil {
		t.Fatal(err)
	}

	proposal2 := primitives.ProposalSignedData{
		Slot:      block2.BlockHeader.SlotNumber,
		Shard:     c.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot2, err := ssz.TreeHash(proposal2)
	if err != nil {
		t.Fatal(err)
	}

	blockSignature2, err := bls.Sign(key, proposalRoot2[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	proposal2DifferentShard := proposal2.Copy()
	proposal2DifferentShard.Shard++

	proposal2DifferentShardHash, err := ssz.TreeHash(proposal2DifferentShard)
	if err != nil {
		t.Fatal(err)
	}

	proposal2DifferentShardSignature, err := bls.Sign(key, proposal2DifferentShardHash[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	proposal2DifferentSlot := proposal2.Copy()
	proposal2DifferentSlot.Slot++

	proposal2DifferentSlotHash, err := ssz.TreeHash(proposal2DifferentSlot)
	if err != nil {
		t.Fatal(err)
	}

	proposal2DifferentSlotSignature, err := bls.Sign(key, proposal2DifferentSlotHash[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	// ProposerSlashings happen when validators sign two blocks at the same slot.

	testBlockBodies := []BodyShouldValidate{
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal1,
				ProposalSignature1: blockSignature1.Serialize(),
				ProposalData2:      proposal2,
				ProposalSignature2: blockSignature2.Serialize(),
			},
			true,
			"should validate with two different signed blocks",
		},
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal1,
				ProposalSignature1: blockSignature1.Serialize(),
				ProposalData2:      proposal1,
				ProposalSignature2: blockSignature1.Serialize(),
			},
			false,
			"should not validate with same signed block",
		},
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal1,
				ProposalSignature1: [48]byte{},
				ProposalData2:      proposal2,
				ProposalSignature2: blockSignature2.Serialize(),
			},
			false,
			"should not validate with invalid signature 1",
		},
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal1,
				ProposalSignature1: blockSignature1.Serialize(),
				ProposalData2:      proposal2,
				ProposalSignature2: [48]byte{},
			},
			false,
			"should not validate with invalid signature 2",
		},
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal2,
				ProposalSignature1: blockSignature2.Serialize(),
				ProposalData2:      proposal2DifferentShard,
				ProposalSignature2: proposal2DifferentShardSignature.Serialize(),
			},
			false,
			"should not validate with different shard",
		},
		{
			primitives.ProposerSlashing{
				ProposerIndex:      proposerIndex,
				ProposalData1:      proposal2,
				ProposalSignature1: blockSignature2.Serialize(),
				ProposalData2:      proposal2DifferentSlot,
				ProposalSignature2: proposal2DifferentSlotSignature.Serialize(),
			},
			false,
			"should not validate with different slot",
		},
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	for _, b := range testBlockBodies {
		blockTest := &primitives.Block{
			BlockHeader: primitives.BlockHeader{
				SlotNumber:   slotToPropose,
				ParentRoot:   chainhash.Hash{},
				StateRoot:    chainhash.Hash{},
				RandaoReveal: [48]byte{},
				Signature:    [48]byte{},
			},
			BlockBody: primitives.BlockBody{
				Attestations: nil,
				ProposerSlashings: []primitives.ProposerSlashing{
					b.ps,
				},
				CasperSlashings: nil,
				Deposits:        nil,
				Exits:           nil,
			},
		}

		err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
		if err != nil {
			t.Fatal(err)
		}

		stateCopy := state.Copy()

		err = stateCopy.ProcessBlock(blockTest, c, FakeBlockView{}, true)

		if (err == nil) != b.validates {
			if b.validates {
				t.Fatalf("expected block to validate, but got error: %s for case: %s", err, b.reason)
			} else {
				t.Fatalf("expected block to fail with reason: %s", b.reason)
			}
		}
	}
}

func TestProposerSlashingPenalty(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, keystore, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	// sign two conflicting blocks at the same height
	slotToPropose := state.Slot + 1
	proposerIndex, err := state.GetBeaconProposerIndex(slotToPropose, c)
	if err != nil {
		t.Fatal(err)
	}

	type BodyShouldValidate struct {
		ps        primitives.ProposerSlashing
		validates bool
		reason    string
	}

	block1 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotToPropose + 1,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: [48]byte{},
			Signature:    [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations:      nil,
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	key := keystore.GetKeyForValidator(proposerIndex)

	var slotBytes [8]byte
	binary.BigEndian.PutUint64(slotBytes[:], block1.BlockHeader.SlotNumber)
	slotBytesHash := chainhash.HashH(slotBytes[:])

	slotBytesSignature, err := bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		t.Fatal(err)
	}

	block1.BlockHeader.RandaoReveal = slotBytesSignature.Serialize()
	block1.BlockHeader.Signature = bls.EmptySignature.Serialize()

	blockWithoutSignatureRoot, err := ssz.TreeHash(block1)
	if err != nil {
		t.Fatal(err)
	}

	proposal1 := primitives.ProposalSignedData{
		Slot:      block1.BlockHeader.SlotNumber,
		Shard:     c.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot1, err := ssz.TreeHash(proposal1)
	if err != nil {
		t.Fatal(err)
	}

	blockSignature1, err := bls.Sign(key, proposalRoot1[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	block2 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotToPropose + 1,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: [48]byte{},
			Signature:    [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations:      []primitives.Attestation{{}},
			ProposerSlashings: nil,
			CasperSlashings:   nil,
			Deposits:          nil,
			Exits:             nil,
		},
	}

	binary.BigEndian.PutUint64(slotBytes[:], block2.BlockHeader.SlotNumber)
	slotBytesHash = chainhash.HashH(slotBytes[:])

	slotBytesSignature, err = bls.Sign(key, slotBytesHash[:], bls.DomainRandao)
	if err != nil {
		t.Fatal(err)
	}

	block2.BlockHeader.RandaoReveal = slotBytesSignature.Serialize()
	block2.BlockHeader.Signature = bls.EmptySignature.Serialize()

	blockWithoutSignatureRoot, err = ssz.TreeHash(block2)
	if err != nil {
		t.Fatal(err)
	}

	proposal2 := primitives.ProposalSignedData{
		Slot:      block2.BlockHeader.SlotNumber,
		Shard:     c.BeaconShardNumber,
		BlockHash: blockWithoutSignatureRoot,
	}

	proposalRoot2, err := ssz.TreeHash(proposal2)
	if err != nil {
		t.Fatal(err)
	}

	blockSignature2, err := bls.Sign(key, proposalRoot2[:], bls.DomainProposal)
	if err != nil {
		t.Fatal(err)
	}

	slashing := primitives.ProposerSlashing{
		ProposerIndex:      proposerIndex,
		ProposalData1:      proposal1,
		ProposalSignature1: blockSignature1.Serialize(),
		ProposalData2:      proposal2,
		ProposalSignature2: blockSignature2.Serialize(),
	}

	err = state.ProcessSlot(chainhash.Hash{}, c)
	if err != nil {
		t.Fatal(err)
	}

	slashedProposerIndex := proposerIndex

	proposerIndex, err = state.GetBeaconProposerIndex(slotToPropose-1, c)
	if err != nil {
		t.Fatal(err)
	}

	blockTest := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   slotToPropose,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: [48]byte{},
			Signature:    [48]byte{},
		},
		BlockBody: primitives.BlockBody{
			Attestations: nil,
			ProposerSlashings: []primitives.ProposerSlashing{
				slashing,
			},
			CasperSlashings: nil,
			Deposits:        nil,
			Exits:           nil,
		},
	}

	err = SignBlock(blockTest, keystore.GetKeyForValidator(proposerIndex), c)
	if err != nil {
		t.Fatal(err)
	}

	stateCopy := state.Copy()

	err = stateCopy.ProcessBlock(blockTest, c, FakeBlockView{}, true)
	if err != nil {
		t.Fatal(err)
	}

	slashedLessBalance := state.ValidatorBalances[slashedProposerIndex] - stateCopy.ValidatorBalances[slashedProposerIndex]

	slashedBalanceBefore := state.GetEffectiveBalance(slashedProposerIndex, c)

	expectedSlash := slashedBalanceBefore / c.WhistleblowerRewardQuotient

	if slashedLessBalance != expectedSlash {
		t.Fatalf("wrong slashed amount. (slashed is %d, should be %d)", slashedLessBalance, expectedSlash)
	}

	whistleblowerMoreBalance := stateCopy.ValidatorBalances[proposerIndex] - state.ValidatorBalances[proposerIndex]

	if whistleblowerMoreBalance != expectedSlash {
		t.Fatalf("wrong whistleblower amount. (whistleblower got %d, should be %d)", whistleblowerMoreBalance, expectedSlash)
	}
}
