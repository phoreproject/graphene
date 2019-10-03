package csmt

import (
	"fmt"
	"github.com/dgraph-io/badger"
	"testing"
)

func TestRandomWritesRollbackCommitBadger(t *testing.T) {
	badgerdb, err := badger.Open(badger.DefaultOptions("./badger-test"))
	if err != nil {
		t.Fatal(err)
	}

	err = badgerdb.DropAll()
	if err != nil {
		t.Fatal(err)
	}

	defer badgerdb.Close()

	under := NewBadgerTreeDB(badgerdb)

	underlyingTree := NewTree(under)

	for i := 0; i < 200; i++ {
		err := underlyingTree.Set(ch(fmt.Sprintf("key%d", i)), ch(fmt.Sprintf("val%d", i)))
		if err != nil {
			t.Fatal(err)
		}
	}

	treeRoot, err := underlyingTree.Hash()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		cachedTreeDB, err := NewTreeTransaction(under)
		if err != nil {
			t.Fatal(err)
		}
		cachedTree := NewTree(cachedTreeDB)

		for newVal := 198; newVal < 202; newVal++ {
			err := cachedTree.Set(ch(fmt.Sprintf("key%d", i)), ch(fmt.Sprintf("val2%d", newVal)))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	underlyingHash, err := underlyingTree.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if !underlyingHash.IsEqual(treeRoot) {
		t.Fatal("expected uncommitted transaction not to affect underlying tree")
	}

	cachedTreeDB, err := NewTreeTransaction(under)
	if err != nil {
		t.Fatal(err)
	}
	cachedTree := NewTree(cachedTreeDB)

	for i := 0; i < 100; i++ {
		for i := 198; i < 202; i++ {
			err := cachedTree.Set(ch(fmt.Sprintf("key%d", i)), ch(fmt.Sprintf("val3%d", i)))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	cachedTreeHash, err := cachedTree.Hash()
	if err != nil {
		t.Fatal(err)
	}

	err = cachedTreeDB.Flush()
	if err != nil {
		t.Fatal(err)
	}

	underlyingHash, err = underlyingTree.Hash()
	if err != nil {
		t.Fatal(err)
	}


	if !cachedTreeHash.IsEqual(underlyingHash) {
		t.Fatal("expected flush to update the underlying tree")
	}

}