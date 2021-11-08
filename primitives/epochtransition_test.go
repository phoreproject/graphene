package primitives_test

import (
	"testing"

	"github.com/prysmaticlabs/go-ssz"

	"github.com/phoreproject/graphene/beacon/config"
	"github.com/phoreproject/graphene/chainhash"
	"github.com/phoreproject/graphene/primitives"
	"github.com/sirupsen/logrus"
)

func generateParticipation(totalValidators uint64, numValidators uint64) []uint8 {
	participation := make([]uint8, (totalValidators+7)/8)

	for i := uint64(0); i < numValidators; i++ {
		participation[i/8] |= 1 << uint(i%8)
	}

	return participation
}

func TestProposalQueueing(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	numValidators := c.ShardCount*c.TargetCommitteeSize*2 + 5

	state, _, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	minValidatorsToPass := (c.QueueThresholdNumerator*uint64(numValidators) + (c.QueueThresholdDenominator - 1)) / c.QueueThresholdDenominator
	minValidatorsToCancel := (c.CancelThresholdNumerator*uint64(numValidators) + (c.CancelThresholdDenominator - 1)) / c.CancelThresholdDenominator

	state.Proposals = []primitives.ActiveProposal{
		{
			Data: primitives.VoteData{
				Type:       primitives.Propose,
				Shards:     []uint32{1, 2, 3},
				ActionHash: chainhash.Hash{1},
				Proposer:   0,
			},
			Participation: generateParticipation(uint64(numValidators), minValidatorsToPass),
			StartEpoch:    0,
			Queued:        false,
		},
		{
			Data: primitives.VoteData{
				Type:       primitives.Propose,
				Shards:     []uint32{1, 2, 3},
				ActionHash: chainhash.Hash{1},
				Proposer:   0,
			},
			Participation: generateParticipation(uint64(numValidators), minValidatorsToPass-1),
			StartEpoch:    0,
			Queued:        false,
		},
	}

	_, err = state.ProcessSlots(c.EpochLength+1, FakeBlockView{}, c)
	if err != nil {
		t.Fatal(err)
	}

	if state.Proposals[0].Queued != true {
		t.Fatal("expected proposal with enough votes to be queued")
	}

	if state.Proposals[1].Queued != false {
		t.Fatal("expected proposal without enough votes to not be queued")
	}

	firstProposalHash, _ := ssz.HashTreeRoot(state.Proposals[0].Data)

	state.Proposals = append(state.Proposals, primitives.ActiveProposal{
		Data: primitives.VoteData{
			Type:       primitives.Cancel,
			Shards:     []uint32{1, 2, 3},
			ActionHash: firstProposalHash,
			Proposer:   0,
		},
		Participation: generateParticipation(uint64(numValidators), minValidatorsToCancel-1),
		StartEpoch:    0,
		Queued:        false,
	})

	state.Proposals = append(state.Proposals, primitives.ActiveProposal{
		Data: primitives.VoteData{
			Type:       primitives.Propose,
			Shards:     []uint32{1, 2, 3, 4},
			ActionHash: chainhash.Hash{1},
			Proposer:   0,
		},
		Participation: generateParticipation(uint64(numValidators), 1),
		StartEpoch:    0,
		Queued:        false,
	})

	_, err = state.ProcessSlots(2*c.EpochLength+1, FakeBlockView{}, c)
	if err != nil {
		t.Fatal(err)
	}

	if state.Proposals[0].Queued != true {
		t.Fatal("expected cancel proposal to not activate without enough votes")
	}

	state.Proposals[2].Participation = generateParticipation(uint64(numValidators), minValidatorsToCancel)

	_, err = state.ProcessSlots(3*c.EpochLength+1, FakeBlockView{}, c)
	if err != nil {
		t.Fatal(err)
	}

	if len(state.Proposals) != 2 {
		t.Fatal("expected queued proposal to be removed when cancelling")
	}

	_, err = state.ProcessSlots(5*c.EpochLength+1, FakeBlockView{}, c)
	if err != nil {
		t.Fatal(err)
	}

	if len(state.Proposals) != 1 {
		t.Fatal("expected proposal to timeout")
	}

	_, err = state.ProcessSlots(6*c.EpochLength+1, FakeBlockView{}, c)
	if err != nil {
		t.Fatal(err)
	}

	if len(state.Proposals) != 0 {
		t.Fatal("expected proposal to timeout")
	}
}
