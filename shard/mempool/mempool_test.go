package mempool

import (
	"testing"
)

func TestShardMempool(t *testing.T) {
	sm := NewShardMempool(ValidateTrue, PrioritizeEqual)

	err := sm.SubmitTransaction([]byte("a"))
	if err != nil {
		t.Fatal(err)
	}

	err = sm.SubmitTransaction([]byte("b"))
	if err != nil {
		t.Fatal(err)
	}

	err = sm.SubmitTransaction([]byte("a"))
	if err != nil {
		t.Fatal(err)
	}

	txs := sm.GetTransactions()
	if len(txs) != 2 {
		t.Fatal(err)
	}

	if (txs[0][0] != 'a' || txs[1][0] != 'b') && (txs[0][0] != 'b' || txs[1][0] != 'a') {
		t.Fatalf("expected GetTransactions to return [\"a\", \"b\"], got: %#v", txs)
	}
}
