package state_test

import (
	"testing"

	"github.com/phoreproject/graphene/chainhash"
	"github.com/phoreproject/graphene/csmt"
	"github.com/phoreproject/graphene/shard/state"
)

func ch(s string) chainhash.Hash {
	return chainhash.HashH([]byte(s))
}

func TestPartialStateGeneration(t *testing.T) {
	treeDB := csmt.NewInMemoryTreeDB()
	tree := csmt.NewTree(treeDB)
	ts, err := state.NewTrackingState(tree)
	if err != nil {
		t.Fatal(err)
	}

	key := ch("test")
	val := ch("test2")

	err = ts.Update(func(a csmt.TreeTransactionAccess) error {
		err = a.Set(key, val)
		if err != nil {
			t.Fatal(err)
		}

		out, err := a.Get(key)
		if err != nil {
			t.Fatal(err)
		}

		if !out.IsEqual(&val) {
			t.Fatal("expected get value to return value set")
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	root, vws, uws := ts.GetWitnesses()

	ps := state.NewPartialShardState(root, vws, uws)

	err = ps.Set(key, val)
	if err != nil {
		t.Fatal(err)
	}

	psVal, err := ps.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	if !psVal.IsEqual(&val) {
		t.Fatal("expected get value from partial tree to match")
	}
}
