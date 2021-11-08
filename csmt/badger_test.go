package csmt

import (
	"fmt"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/phoreproject/graphene/chainhash"
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

	var treeRoot chainhash.Hash

	err = underlyingTree.Update(func(tx TreeTransactionAccess) error {
		for i := 0; i < 200; i++ {
			err := tx.Set(ch(fmt.Sprintf("key%d", i)), ch(fmt.Sprintf("val%d", i)))
			if err != nil {
				return err
			}
		}

		initialRoot, err := tx.Hash()
		if err != nil {
			return err
		}

		treeRoot = *initialRoot

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		cachedTreeDB, err := NewTreeMemoryCache(under)
		if err != nil {
			t.Fatal(err)
		}
		cachedTree := NewTree(cachedTreeDB)

		err = cachedTree.Update(func(tx TreeTransactionAccess) error {
			for newVal := 198; newVal < 202; newVal++ {
				err := tx.Set(ch(fmt.Sprintf("key%d", i)), ch(fmt.Sprintf("val2%d", newVal)))
				if err != nil {
					t.Fatal(err)
				}
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	var underlyingHash chainhash.Hash

	err = underlyingTree.View(func(tx TreeTransactionAccess) error {
		h, err := tx.Hash()
		if err != nil {
			return err
		}
		underlyingHash = *h

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if !underlyingHash.IsEqual(&treeRoot) {
		t.Fatal("expected uncommitted transaction not to affect underlying tree")
	}

	cachedTreeDB, err := NewTreeMemoryCache(under)
	if err != nil {
		t.Fatal(err)
	}
	cachedTree := NewTree(cachedTreeDB)

	err = cachedTree.Update(func(tx TreeTransactionAccess) error {
		for i := 0; i < 100; i++ {
			for newVal := 198; newVal < 202; newVal++ {
				err := tx.Set(ch(fmt.Sprintf("key%d", i)), ch(fmt.Sprintf("val3%d", newVal)))
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var cachedTreeHash chainhash.Hash

	err = cachedTree.View(func(tx TreeTransactionAccess) error {
		h, err := tx.Hash()
		if err != nil {
			return err
		}
		cachedTreeHash = *h

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = cachedTreeDB.Flush()
	if err != nil {
		t.Fatal(err)
	}

	err = underlyingTree.View(func(tx TreeTransactionAccess) error {
		h, err := tx.Hash()
		if err != nil {
			return err
		}
		underlyingHash = *h

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if !cachedTreeHash.IsEqual(&underlyingHash) {
		t.Fatal("expected flush to update the underlying tree")
	}
}
