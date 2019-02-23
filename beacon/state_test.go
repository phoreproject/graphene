package beacon_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/internal/util"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

func TestLastBlockOnInitialSetup(t *testing.T) {
	b, keys, err := util.SetupBlockchain(config.RegtestConfig.ShardCount*config.RegtestConfig.TargetCommitteeSize*2+1, &config.RegtestConfig)
	if err != nil {
		t.Fatal(err)
	}

	b0, err := b.LastBlock()
	if err != nil {
		t.Fatal(err)
	}

	if b0.BlockHeader.SlotNumber != 0 {
		t.Fatal("invalid last block for initial chain")
	}

	s := b.GetState()

	proposerIndex, err := s.GetBeaconProposerIndex(s.Slot, 0, b.GetConfig())
	if err != nil {
		t.Fatal(err)
	}

	_, err = util.MineBlockWithFullAttestations(b, keys, proposerIndex)
	if err != nil {
		t.Fatal(err)
	}

	if b.Height() != 1 {
		t.Fatalf("invalid height (expected: 1, got: %d)", b.Height())
	}

	b1, err := b.LastBlock()
	if err != nil {
		t.Fatal(err)
	}

	if b1.BlockHeader.SlotNumber != 1 {
		t.Fatal("invalid last block after mining 1 block")
	}
}

func TestStateInitialization(t *testing.T) {
	b, keys, err := util.SetupBlockchain(config.RegtestConfig.ShardCount*config.RegtestConfig.TargetCommitteeSize*2+1, &config.RegtestConfig)
	if err != nil {
		t.Fatal(err)
	}

	s := b.GetState()
	proposerIndex, err := s.GetBeaconProposerIndex(s.Slot, 0, b.GetConfig())
	if err != nil {
		t.Fatal(err)
	}

	_, err = util.MineBlockWithFullAttestations(b, keys, proposerIndex)
	if err != nil {
		t.Fatal(err)
	}

	s = b.GetState()

	if len(s.ShardAndCommitteeForSlots[0]) == 0 {
		t.Errorf("invalid initial validator entries")
	}

	if len(s.ShardAndCommitteeForSlots) != int(config.RegtestConfig.EpochLength*2) {
		t.Errorf("shardandcommitteeforslots array is not big enough (got: %d, expected: %d)", len(s.ShardAndCommitteeForSlots), config.RegtestConfig.EpochLength)
	}
}

func TestCrystallizedStateTransition(t *testing.T) {
	b, keys, err := util.SetupBlockchain(config.RegtestConfig.ShardCount*config.RegtestConfig.TargetCommitteeSize*2+5, &config.RegtestConfig)
	if err != nil {
		t.Fatal(err)
	}

	firstValidator := b.GetState().ShardAndCommitteeForSlots[0][0].Committee[0]

	for i := uint64(0); i < uint64(b.GetConfig().EpochLength)*5; i++ {
		s := b.GetState()
		proposerIndex, err := s.GetBeaconProposerIndex(s.Slot, i, b.GetConfig())
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("proposer %d mining block %d\n", proposerIndex, i+1)
		_, err = util.MineBlockWithFullAttestations(b, keys, proposerIndex)
		if err != nil {
			t.Fatal(err)
		}

		s = b.GetState()

		fmt.Printf("justified slot: %d, finalized slot: %d, justificationBitField: %b, previousJustifiedSlot: %d\n", s.JustifiedSlot, s.FinalizedSlot, s.JustificationBitfield, s.PreviousJustifiedSlot)
	}

	timer := time.NewTimer(time.Until(b.GetNextSlotTime().Add(time.Millisecond * 500)))
	<-timer.C

	stateAfterSlot20 := b.GetState()

	firstValidator2 := stateAfterSlot20.ShardAndCommitteeForSlots[0][0].Committee[0]
	if firstValidator == firstValidator2 {
		t.Fatal("validators were not shuffled")
	}
	if stateAfterSlot20.FinalizedSlot != 12 || stateAfterSlot20.JustifiedSlot != 16 || stateAfterSlot20.JustificationBitfield != 31 || stateAfterSlot20.PreviousJustifiedSlot != 12 {
		t.Fatal("justification/finalization is working incorrectly")
	}
}
