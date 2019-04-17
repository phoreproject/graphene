package beacon_test

import (
	"fmt"
	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/chainhash"
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
	if stateAfterSlot20.FinalizedSlot != 8 || stateAfterSlot20.JustifiedSlot != 12 || stateAfterSlot20.JustificationBitfield != 7 || stateAfterSlot20.PreviousJustifiedSlot != 8 {
		t.Fatal("justification/finalization is working incorrectly")
	}
}

func TestSlotTransition(t *testing.T) {
	b, _, err := util.SetupBlockchain(config.RegtestConfig.ShardCount*config.RegtestConfig.TargetCommitteeSize*2+5, &config.RegtestConfig)
	if err != nil {
		t.Fatal(err)
	}

	logrus.SetLevel(logrus.ErrorLevel)

	state := b.GetState()

	for i := 0; i < 1000; i++ {
		newState := state.Copy()

		err = newState.ProcessSlot(b.Tip(), &config.RegtestConfig)
		if err != nil {
			t.Fatal(err)
		}

		if newState.Slot != state.Slot+1 {
			t.Fatal("expected slot to be incremented on slot transition")
		}

		tip := b.Tip()

		if !newState.LatestBlockHashes[(newState.Slot-1)%config.RegtestConfig.LatestBlockRootsLength].IsEqual(&tip) {
			t.Fatalf("expected latest block hashes to be updated on slot transition (expected: %d, got: %d)",
				tip,
				newState.LatestBlockHashes[(newState.Slot-1)%config.RegtestConfig.LatestBlockRootsLength])
		}

		if newState.Slot%config.RegtestConfig.LatestBlockRootsLength == 0 {
			h, err := ssz.TreeHash(newState.LatestBlockHashes)
			if err != nil {
				t.Fatal(err)
			}
			ch := chainhash.Hash(h)

			if !newState.BatchedBlockRoots[len(newState.BatchedBlockRoots)-1].IsEqual(&ch) {
				t.Fatalf("expected batched block roots to be updated on slot transition (expected: %d, got: %d)",
					ch,
					newState.BatchedBlockRoots[len(newState.BatchedBlockRoots)-1])
			}
		}

		state = newState
	}

}
