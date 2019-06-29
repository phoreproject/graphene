package primitives_test

import (
	"testing"

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
