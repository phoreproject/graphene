package beacon_test

import (
	"os"
	"testing"
	"time"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/sirupsen/logrus"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/util"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

func TestStateInitialization(t *testing.T) {
	logrus.SetLevel(logrus.ErrorLevel)

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
	logrus.SetLevel(logrus.DebugLevel)

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
		_, err = util.MineBlockWithFullAttestations(b, keys, proposerIndex)
		if err != nil {
			t.Fatal(err)
		}

		s = b.GetState()
	}
	stateAfterSlot20 := b.GetState()

	firstValidator2 := stateAfterSlot20.ShardAndCommitteeForSlots[0][0].Committee[0]
	if firstValidator == firstValidator2 {
		t.Fatal("validators were not shuffled")
	}
	if stateAfterSlot20.FinalizedSlot != 12 || stateAfterSlot20.JustifiedSlot != 16 || stateAfterSlot20.JustificationBitfield != 31 || stateAfterSlot20.PreviousJustifiedSlot != 12 {
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

		err = newState.ProcessSlot(b.View.Chain.Tip().Hash, &config.RegtestConfig)
		if err != nil {
			t.Fatal(err)
		}

		if newState.Slot != state.Slot+1 {
			t.Fatal("expected slot to be incremented on slot transition")
		}

		tip := b.View.Chain.Tip().Hash

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

func BenchmarkSlotTransition(t *testing.B) {
	b, _, err := util.SetupBlockchain(config.RegtestConfig.ShardCount*config.RegtestConfig.TargetCommitteeSize*2+5, &config.RegtestConfig)
	if err != nil {
		t.Fatal(err)
	}

	logrus.SetLevel(logrus.ErrorLevel)

	state := b.GetState()

	t.ResetTimer()

	for i := 0; i < t.N; i++ {
		newState := state.Copy()

		err = newState.ProcessSlot(b.View.Chain.Tip().Hash, &config.RegtestConfig)
		if err != nil {
			t.Fatal(err)
		}

		state = newState
	}
}

func BenchmarkBlockTransition(t *testing.B) {
	//logrus.SetLevel(logrus.ErrorLevel)

	genesisTime := time.Now()

	b, keys, err := util.SetupBlockchainWithTime(config.RegtestConfig.ShardCount*config.RegtestConfig.TargetCommitteeSize*2+5, &config.RegtestConfig, genesisTime)
	if err != nil {
		t.Fatal(err)
	}

	for i := uint64(0); i < uint64(t.N); i++ {
		s := b.GetState()
		proposerIndex, err := s.GetBeaconProposerIndex(s.Slot, i, b.GetConfig())
		if err != nil {
			t.Fatal(err)
		}
		_, err = util.MineBlockWithFullAttestations(b, keys, proposerIndex)
		if err != nil {
			t.Fatal(err)
		}

		s = b.GetState()
	}

	blocks := make([]primitives.Block, b.Height())

	for i := range blocks {
		nodeAtSlot, err := b.View.Chain.GetBlockBySlot(uint64(i + 1))
		if err != nil {
			t.Fatal(err)
		}

		blockAtSlot, err := b.GetBlockByHash(nodeAtSlot.Hash)

		blocks[i] = *blockAtSlot
	}

	b0, _, err := util.SetupBlockchainWithTime(config.RegtestConfig.ShardCount*config.RegtestConfig.TargetCommitteeSize*2+5, &config.RegtestConfig, genesisTime)
	if err != nil {
		t.Fatal(err)
	}

	t.ResetTimer()

	for i := range blocks {
		_, _, err := b0.ProcessBlock(&blocks[i], false, true)
		if err != nil {
			t.Fatal(err)
		}
	}
}
