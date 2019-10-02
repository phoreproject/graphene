package csmt

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/phoreproject/synapse/chainhash"
)

func ch(s string) chainhash.Hash {
	return chainhash.HashH([]byte(s))
}

func TestTree_RandomSet(t *testing.T) {
	keys := make([]chainhash.Hash, 500)
	val := ch("testval")
	tree := NewTree(NewInMemoryTreeDB(), NewInMemoryKVStore())

	for i := range keys {
		keys[i] = ch(fmt.Sprintf("%d", i))
	}

	for i := 0; i < 500; i++ {
		tree.Set(keys[i], val)
	}
}

func TestTree_SetZero(t *testing.T) {
	val := emptyHash

	tree := NewTree(NewInMemoryTreeDB(), NewInMemoryKVStore())

	tree.Set(ch("1"), val)
	tree.Set(ch("2"), val)
	tree.Set(ch("3"), val)
	tree.Set(ch("4"), val)
	tree.Set(ch("5"), val)
	th := tree.Hash()

	if !th.IsEqual(&emptyTrees[255]) {
		t.Fatalf("expected tree to match %s but got %s", emptyTrees[255], th)
	}
}

func BenchmarkTree_Set(b *testing.B) {
	keys := make([]chainhash.Hash, b.N)
	val := ch("testval")
	t := NewTree(NewInMemoryTreeDB(), NewInMemoryKVStore())

	for i := range keys {
		keys[i] = ch(fmt.Sprintf("%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t.Set(keys[i], val)
	}
}

func Test_calculateSubtreeHashWithOneLeaf(t *testing.T) {
	type args struct {
		key     chainhash.Hash
		value   chainhash.Hash
		atLevel uint8
	}
	tests := []struct {
		name string
		args args
		want chainhash.Hash
	}{
		{
			name: "test lowest node",
			args: args{
				key:     ch("test"),
				value:   emptyHash,
				atLevel: 0,
			},
			want: emptyHash,
		},
		{
			name: "test empty subtree root",
			args: args{
				key:     ch("test"),
				value:   emptyHash,
				atLevel: 255,
			},
			want: emptyTrees[255],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateSubtreeHashWithOneLeaf(&tt.args.key, &tt.args.value, tt.args.atLevel); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calculateSubtreeHashWithOneLeaf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRandomGenerateUpdateWitness(t *testing.T) {
	keys := make([]chainhash.Hash, 500)
	val := ch("testval")
	treeDB := NewInMemoryTreeDB()
	treeKV := NewInMemoryKVStore()
	tree := NewTree(treeDB, treeKV)

	for i := range keys {
		keys[i] = ch(fmt.Sprintf("%d", i))
	}

	for i := 0; i < 2; i++ {
		tree.Set(keys[i], val)
	}

	treehash := tree.Hash()

	for i := 0; i < 1; i++ {
		w := GenerateUpdateWitness(treeDB, treeKV, keys[i], val)
		root, err := CalculateRoot(keys[i], val, w.WitnessBitfield, w.Witnesses, w.LastLevel)
		if err != nil {
			t.Fatal(err)
		}
		if !root.IsEqual(&treehash) {
			t.Fatalf("expected witness root to equal tree hash (expected: %s, got: %s)", treehash, root)
		}
	}
}

func TestGenerateUpdateWitnessEmptyTree(t *testing.T) {
	treeDB := NewInMemoryTreeDB()
	treeKV := NewInMemoryKVStore()
	tree := NewTree(treeDB, treeKV)

	w := GenerateUpdateWitness(treeDB, treeKV, ch("asdf"), ch("1"))

	newRoot, err := w.Apply(tree.Hash())
	if err != nil {
		t.Fatal(err)
	}

	tree.Set(ch("asdf"), ch("1"))

	th := tree.Hash()
	if !th.IsEqual(newRoot) {
		t.Fatalf("expected calculated state root (%s) to match tree state root (%s)", newRoot, th)
	}
}

func TestGenerateUpdateWitnessUpdate(t *testing.T) {
	treeDB := NewInMemoryTreeDB()
	treeKV := NewInMemoryKVStore()
	tree := NewTree(treeDB, treeKV)

	tree.Set(ch("asdf"), ch("2"))
	tree.Set(ch("asdf1"), ch("2"))
	//tree.Set(ch("asdf2"), ch("2"12))
	//tree.Set(ch("asdf3"), ch("2"))
	//tree.Set(ch("asdf4"), ch("2"))

	for i := 0; i < 1; i++ {
		setVal := fmt.Sprintf("%d", i)

		w := GenerateUpdateWitness(treeDB, treeKV, ch("asdf"), ch(setVal))

		newRoot, err := w.Apply(tree.Hash())
		if err != nil {
			t.Fatal(err)
		}

		tree.Set(ch("asdf"), ch(setVal))

		th := tree.Hash()
		if !th.IsEqual(newRoot) {
			t.Fatalf("expected calculated state root (%s) to match tree state root (%s)", newRoot, th)
		}
	}
}

func BenchmarkGenerateUpdateWitness(b *testing.B) {
	keys := make([]chainhash.Hash, b.N)
	val := ch("testval")
	treeDB := NewInMemoryTreeDB()
	treeKV := NewInMemoryKVStore()
	tree := NewTree(treeDB, treeKV)

	for i := range keys {
		keys[i] = ch(fmt.Sprintf("%d", i))
	}

	for i := 0; i < b.N; i++ {
		tree.Set(keys[i], val)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		GenerateUpdateWitness(treeDB, treeKV, keys[i], val)
	}
}

func TestChainedUpdates(t *testing.T) {
	tree := NewTree(NewInMemoryTreeDB(), NewInMemoryKVStore())

	initialRoot := tree.Hash()
	witnesses := make([]*UpdateWitness, 0)

	// start by generating a bunch of witnesses
	for i := 0; i < 1000; i++ {
		key := ch(fmt.Sprintf("key%d", i))

		// test empty key
		testProof2 := tree.Prove(key)
		if testProof2.Check(tree.Hash()) == false {
			t.Fatal("expected verification witness to verify")
		}

		val := ch(fmt.Sprintf("val%d", i))

		witnesses = append(witnesses, tree.SetWithWitness(key, val))

		testProof := tree.Prove(key)
		if testProof.Check(tree.Hash()) == false {
			t.Fatal("expected verification witness to verify")
		}
	}

	// then update half of them
	for i := 0; i < 500; i++ {
		key := ch(fmt.Sprintf("key%d", i))

		// test empty key
		testProof2 := tree.Prove(key)
		if testProof2.Check(tree.Hash()) == false {
			t.Fatal("expected verification witness to verify")
		}

		val := ch(fmt.Sprintf("val1%d", i))

		witnesses = append(witnesses, tree.SetWithWitness(key, val))

		testProof := tree.Prove(key)
		if testProof.Check(tree.Hash()) == false {
			t.Fatal("expected verification witness to verify")
		}
	}

	currentRoot := initialRoot
	for i := range witnesses {
		newRoot, err := witnesses[i].Apply(currentRoot)
		if err != nil {
			t.Fatal(err)
		}
		currentRoot = *newRoot
	}

	th := tree.Hash()
	if !th.IsEqual(&currentRoot) {
		t.Fatal("expected hash after applying updates to match")
	}
}

func TestEmptyBranchWitness(t *testing.T) {
	tree := NewTree(NewInMemoryTreeDB(), NewInMemoryKVStore())
	preroot := tree.Hash()

	w0 := tree.SetWithWitness(ch("test"), ch("asdf"))

	w1 := tree.SetWithWitness(ch("asdfghi"), ch("asdf"))

	newRoot, err := w0.Apply(preroot)
	if err != nil {
		t.Fatal(err)
	}

	newRoot, err = w1.Apply(*newRoot)
	if err != nil {
		t.Fatal(err)
	}

}

func TestCheckWitness(t *testing.T) {
	tree := NewTree(NewInMemoryTreeDB(), NewInMemoryKVStore())
	//preroot := tree.Hash()

	tree.Set(ch("test"), ch("asdf"))
	tree.Set(ch("asdfghi"), ch("asdf"))

	testProof := tree.Prove(ch("test"))
	if testProof.Check(tree.Hash()) == false {
		t.Fatal("expected verification witness to verify")
	}

	// test empty key
	testProof2 := tree.Prove(ch("test1"))
	if testProof2.Check(tree.Hash()) == false {
		t.Fatal("expected verification witness to verify")
	}
}
