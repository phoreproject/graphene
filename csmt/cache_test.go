package csmt

import (
	"fmt"
	"testing"
)

func TestRandomWritesRollbackCommit(t *testing.T) {
	under := NewInMemoryTreeDB()
	kv := NewInMemoryKVStore()

	underlyingTree := NewTree(under, kv)

	for i := 0; i < 200; i++ {
		underlyingTree.Set(ch(fmt.Sprintf("key%d", i)), ch(fmt.Sprintf("val%d", i)))
	}

	treeRoot := underlyingTree.Hash()

	for i := 0; i < 100; i++ {
		cachedTreeDB := NewTreeTransaction(under, kv)
		cachedTree := NewTree(&cachedTreeDB, &cachedTreeDB)

		for i := 198; i < 202; i++ {
			cachedTree.Set(ch(fmt.Sprintf("key%d", i)), ch(fmt.Sprintf("val2%d", i)))
		}
	}

	underlyingHash := underlyingTree.Hash()

	if !underlyingHash.IsEqual(&treeRoot) {
		t.Fatal("expected uncommitted transaction not to affect underlying tree")
	}

	for i := 0; i < 100; i++ {
		cachedTreeDB := NewTreeTransaction(under, kv)
		cachedTree := NewTree(&cachedTreeDB, &cachedTreeDB)

		for i := 198; i < 202; i++ {
			cachedTree.Set(ch(fmt.Sprintf("key%d", i)), ch(fmt.Sprintf("val3%d", i)))
		}

		cachedTreeDB.Flush()

		underlyingHash := underlyingTree.Hash()
		cachedTreeHash := cachedTree.Hash()

		if !cachedTreeHash.IsEqual(&underlyingHash) {
			t.Fatal("expected flush to update the underlying tree")
		}
	}
}