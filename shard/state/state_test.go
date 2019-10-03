package state_test

import (
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/shard/state"
	"testing"
)

func ch(s string) chainhash.Hash {
	return chainhash.HashH([]byte(s))
}

func TestPartialStateGeneration(t *testing.T) {
	fs := state.NewFullShardState(csmt.NewInMemoryTreeDB())
	ts := state.NewTrackingState(fs)

	key := ch("test")
	val := ch("test2")

	err := ts.Set(key, val)
	if err != nil {
		t.Fatal(err)
	}

	out, err := ts.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	if !out.IsEqual(&val) {
		t.Fatal("expected get value to return value set")
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
