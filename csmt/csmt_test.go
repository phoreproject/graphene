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
	tree := NewTree(NewInMemoryTreeDB())

	for i := range keys {
		keys[i] = ch(fmt.Sprintf("%d", i))
	}

	for i := 0; i < 500; i++ {
		err := tree.Set(keys[i], val)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTree_SetZero(t *testing.T) {
	val := emptyHash

	tree := NewTree(NewInMemoryTreeDB())

	err := tree.Set(ch("1"), val)
	if err != nil {
		t.Fatal(err)
	}
	err = tree.Set(ch("2"), val)
	if err != nil {
		t.Fatal(err)
	}
	err = tree.Set(ch("3"), val)
	if err != nil {
		t.Fatal(err)
	}
	err = tree.Set(ch("4"), val)
	if err != nil {
		t.Fatal(err)
	}
	err = tree.Set(ch("5"), val)
	if err != nil {
		t.Fatal(err)
	}
	th, err := tree.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if !th.IsEqual(&emptyTrees[255]) {
		t.Fatalf("expected tree to match %s but got %s", emptyTrees[255], th)
	}
}

func BenchmarkTree_Set(b *testing.B) {
	keys := make([]chainhash.Hash, b.N)
	val := ch("testval")
	t := NewTree(NewInMemoryTreeDB())

	for i := range keys {
		keys[i] = ch(fmt.Sprintf("%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := t.Set(keys[i], val)
		if err != nil {
			b.Fatal(err)
		}
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
	tree := NewTree(treeDB)

	for i := range keys {
		keys[i] = ch(fmt.Sprintf("%d", i))
	}

	for i := 0; i < 2; i++ {
		err := tree.Set(keys[i], val)
		if err != nil {
			t.Fatal(err)
		}
	}

	treehash, err := tree.Hash()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1; i++ {
		w, err := GenerateUpdateWitness(treeDB, keys[i], val)
		if err != nil {
			t.Fatal(err)
		}
		root, err := CalculateRoot(keys[i], val, w.WitnessBitfield, w.Witnesses, w.LastLevel)
		if err != nil {
			t.Fatal(err)
		}
		if !root.IsEqual(treehash) {
			t.Fatalf("expected witness root to equal tree hash (expected: %s, got: %s)", treehash, root)
		}
	}
}

func TestGenerateUpdateWitnessEmptyTree(t *testing.T) {
	treeDB := NewInMemoryTreeDB()
	tree := NewTree(treeDB)

	w, err := GenerateUpdateWitness(treeDB, ch("asdf"), ch("1"))
	if err != nil {
		t.Fatal(err)
	}

	th, err := tree.Hash()
	if err != nil {
		t.Fatal(err)
	}

	newRoot, err := w.Apply(*th)
	if err != nil {
		t.Fatal(err)
	}

	err = tree.Set(ch("asdf"), ch("1"))
	if err != nil {
		t.Fatal(err)
	}

	th, err = tree.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if !th.IsEqual(newRoot) {
		t.Fatalf("expected calculated state root (%s) to match tree state root (%s)", newRoot, th)
	}
}

func TestGenerateUpdateWitnessUpdate(t *testing.T) {
	treeDB := NewInMemoryTreeDB()
	tree := NewTree(treeDB)

	err := tree.Set(ch("asdf"), ch("2"))
	if err != nil {
		t.Fatal(err)
	}
	err = tree.Set(ch("asdf1"), ch("2"))
	if err != nil {
		t.Fatal(err)
	}
	err = tree.Set(ch("asdf2"), ch("2"))
	if err != nil {
		t.Fatal(err)
	}
	err = tree.Set(ch("asdf3"), ch("2"))
	if err != nil {
		t.Fatal(err)
	}
	err = tree.Set(ch("asdf4"), ch("2"))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1; i++ {
		setVal := fmt.Sprintf("%d", i)

		w, err := GenerateUpdateWitness(treeDB, ch("asdf"), ch(setVal))
		if err != nil {
			t.Fatal(err)
		}

		th, err := tree.Hash()
		if err != nil {
			t.Fatal(err)
		}

		newRoot, err := w.Apply(*th)
		if err != nil {
			t.Fatal(err)
		}

		err = tree.Set(ch("asdf"), ch(setVal))
		if err != nil {
			t.Fatal(err)
		}

		th, err = tree.Hash()
		if err != nil {
			t.Fatal(err)
		}
		if !th.IsEqual(newRoot) {
			t.Fatalf("expected calculated state root (%s) to match tree state root (%s)", newRoot, th)
		}
	}
}

func BenchmarkGenerateUpdateWitness(b *testing.B) {
	keys := make([]chainhash.Hash, b.N)
	val := ch("testval")
	treeDB := NewInMemoryTreeDB()
	tree := NewTree(treeDB)

	for i := range keys {
		keys[i] = ch(fmt.Sprintf("%d", i))
	}

	for i := 0; i < b.N; i++ {
		err := tree.Set(keys[i], val)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := GenerateUpdateWitness(treeDB, keys[i], val)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestChainedUpdates(t *testing.T) {
	tree := NewTree(NewInMemoryTreeDB())

	initialRoot, err := tree.Hash()
	if err != nil {
		t.Fatal(err)
	}
	witnesses := make([]*UpdateWitness, 0)

	// start by generating a bunch of witnesses
	for i := 0; i < 1000; i++ {
		key := ch(fmt.Sprintf("key%d", i))

		// test empty key
		testProof2, err := tree.Prove(key)
		if err != nil {
			t.Fatal(err)
		}

		th, err := tree.Hash()
		if err != nil {
			t.Fatal(err)
		}
		if testProof2.Check(*th) == false {
			t.Fatal("expected verification witness to verify")
		}

		val := ch(fmt.Sprintf("val%d", i))

		uw, err := tree.SetWithWitness(key, val)
		if err != nil {
			t.Fatal(err)
		}
		witnesses = append(witnesses, uw)

		testProof, err := tree.Prove(key)
		if err != nil {
			t.Fatal(err)
		}
		th, err = tree.Hash()
		if err != nil {
			t.Fatal(err)
		}
		if testProof.Check(*th) == false {
			t.Fatal("expected verification witness to verify")
		}
	}

	// then update half of them
	for i := 0; i < 500; i++ {
		key := ch(fmt.Sprintf("key%d", i))

		// test empty key
		testProof2, err := tree.Prove(key)
		if err != nil {
			t.Fatal(err)
		}
		th, err := tree.Hash()
		if err != nil {
			t.Fatal(err)
		}
		if testProof2.Check(*th) == false {
			t.Fatal("expected verification witness to verify")
		}

		val := ch(fmt.Sprintf("val1%d", i))

		uw, err := tree.SetWithWitness(key, val)
		if err != nil {
			t.Fatal(err)
		}

		witnesses = append(witnesses, uw)

		testProof, err := tree.Prove(key)
		if err != nil {
			t.Fatal(err)
		}
		th, err = tree.Hash()
		if err != nil {
			t.Fatal(err)
		}
		if testProof.Check(*th) == false {
			t.Fatal("expected verification witness to verify")
		}
	}

	currentRoot := initialRoot
	for i := range witnesses {
		newRoot, err := witnesses[i].Apply(*currentRoot)
		if err != nil {
			t.Fatal(err)
		}
		currentRoot = newRoot
	}

	th, err := tree.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if !th.IsEqual(currentRoot) {
		t.Fatal("expected hash after applying updates to match")
	}
}

func TestEmptyBranchWitness(t *testing.T) {
	tree := NewTree(NewInMemoryTreeDB())
	preroot, err := tree.Hash()
	if err != nil {
		t.Fatal(err)
	}

	w0, err := tree.SetWithWitness(ch("test"), ch("asdf"))
	if err != nil {
		t.Fatal(err)
	}

	w1, err := tree.SetWithWitness(ch("asdfghi"), ch("asdf"))
	if err != nil {
		t.Fatal(err)
	}

	newRoot, err := w0.Apply(*preroot)
	if err != nil {
		t.Fatal(err)
	}

	newRoot, err = w1.Apply(*newRoot)
	if err != nil {
		t.Fatal(err)
	}

}

func TestCheckWitness(t *testing.T) {
	tree := NewTree(NewInMemoryTreeDB())
	//preroot := tree.Hash()

	err := tree.Set(ch("test"), ch("asdf"))
	if err != nil {
		t.Fatal(err)
	}
	err = tree.Set(ch("asdfghi"), ch("asdf"))
	if err != nil {
		t.Fatal(err)
	}

	testProof, err := tree.Prove(ch("test"))
	if err != nil {
		t.Fatal(err)
	}
	th, err := tree.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if testProof.Check(*th) == false {
		t.Fatal("expected verification witness to verify")
	}

	// test empty key
	testProof2, err := tree.Prove(ch("test1"))
	if err != nil {
		t.Fatal(err)
	}
	th, err = tree.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if testProof2.Check(*th) == false {
		t.Fatal("expected verification witness to verify")
	}
}
