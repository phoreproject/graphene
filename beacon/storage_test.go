package beacon_test

import (
	"github.com/go-test/deep"
	"testing"

	"github.com/phoreproject/graphene/beacon"
	"github.com/phoreproject/graphene/beacon/config"
	"github.com/phoreproject/graphene/beacon/util"
	"github.com/sirupsen/logrus"
)

func TestEmptyStateStore(t *testing.T) {
	logrus.SetLevel(logrus.ErrorLevel)
	numValidators := config.RegtestConfig.ShardCount*config.RegtestConfig.TargetCommitteeSize*2 + 1

	b, keys, err := util.SetupBlockchain(numValidators, &config.RegtestConfig)
	if err != nil {
		t.Fatal(err)
	}

	for i := uint64(0); i < uint64(b.GetConfig().EpochLength)*5+1; i++ {
		s := b.GetState()

		if s.EpochIndex != i/b.GetConfig().EpochLength {
			s = s.Copy()

			_, err = s.ProcessEpochTransition(b.GetConfig())
			if err != nil {
				t.Fatal(err)
			}
		}

		proposerIndex, err := s.GetBeaconProposerIndex(i, b.GetConfig())
		if err != nil {
			t.Fatal(err)
		}
		_, err = util.MineBlockWithFullAttestations(b, keys, proposerIndex)
		if err != nil {
			t.Fatal(err)
		}
	}

	validators, err := util.InitialValidators(numValidators, keys, &config.RegtestConfig)
	if err != nil {
		t.Fatal(err)
	}

	b2, err := beacon.NewBlockchainWithInitialValidators(b.DB, &config.RegtestConfig, validators, true, b.GetGenesisTime())
	if err != nil {
		t.Fatal(err)
	}

	node := b2.View.Chain.Tip()
	for node != nil {
		n2 := b.View.Chain.GetBlockBySlot(node.Slot)
		if !n2.Hash.IsEqual(&node.Hash) {
			t.Fatal("expected chain to match")
		}

		node = node.Parent
	}

	if diff := deep.Equal(b.GetState(), b2.GetState()); diff != nil {
		t.Fatal(diff)
	}
}
