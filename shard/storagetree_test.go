package shard

import (
	"testing"

	"github.com/phoreproject/synapse/chainhash"
)

var hashNull = chainhash.Hash{}

func newHashFromStr(s string) *chainhash.Hash {
	result, _ := chainhash.NewHashFromStr(s)
	return result
}

func buildStorageTree(keys []string) *StorageTree {
	tree := NewStorageTree()
	for _, k := range keys {
		h := newHashFromStr(k)
		tree.Set(*h, hashNull)
	}
	return tree
}

/*
func TestStorageTree_computeHash(t *testing.T) {
	keys := []string{"01", "02", "03", "04", "05"}

	tree := buildStorageTree(keys)
	rootHash := tree.Hash()
	t.Logf("Original tree")
	t.Logf("%s", tree.merkleTree.DebugToJSONString())

	var witnesses []Witness

	var wt Witness
	var err error

	wt, err = tree.SetWithWitness(*newHashFromStr("10"), hashNull)
	if err != nil {
		t.Fatal(err)
	}
	witnesses = append(witnesses, wt)
	t.Logf("Witness set-10: %s", wt.proof.DebugToString())
	t.Logf("Tree after set-10")
	t.Logf("%s", tree.merkleTree.DebugToJSONString())

	wt, err = tree.SetWithWitness(*newHashFromStr("11"), hashNull)
	if err != nil {
		t.Fatal(err)
	}
	witnesses = append(witnesses, wt)
	t.Logf("Witness set-11: %s", wt.proof.DebugToString())
	t.Logf("Tree after set-11")
	t.Logf("%s", tree.merkleTree.DebugToJSONString())

	wt, err = tree.DeleteWithWitness(*newHashFromStr("01"))
	if err != nil {
		t.Fatal(err)
	}
	witnesses = append(witnesses, wt)
	t.Logf("Witness delete-01: %s", wt.proof.DebugToString())
	t.Logf("Tree after Delete-01")
	t.Logf("%s", tree.merkleTree.DebugToJSONString())

	ptree := NewPartialTree(rootHash, witnesses)
	ptree.Set(*newHashFromStr("10"), hashNull)
	ptree.Set(*newHashFromStr("11"), hashNull)
	ptree.Delete(*newHashFromStr("01"))

	newRootHash := tree.Hash()
	t.Logf("New root hash %s\n", newRootHash.String())

	newHash, _ := ptree.computeHash()
	t.Logf("New hash %s\n", newHash.String())

	t.Fatal("OK")
}
*/

type testItem struct {
	operation int
	key       string
}

func doTestStorageTreeGeneral(t *testing.T, name string, keys []string, items []testItem) {
	tree := buildStorageTree(keys)
	rootHash := tree.Hash()

	t.Logf("Original tree")
	t.Logf("%s", tree.merkleTree.DebugToJSONString())

	var witnesses []Witness
	var wt Witness
	var err error
	for _, item := range items {
		if item.operation == witOpSet {
			wt, err = tree.SetWithWitness(*newHashFromStr(item.key), hashNull)
			if err != nil {
				t.Fatal(err)
			}
			witnesses = append(witnesses, wt)

			t.Logf("Witness set-%s: %s", item.key, wt.proof.DebugToString())

			t.Logf("Tree set witness-%s", item.key)
			t.Logf("%s", tree.merkleTree.DebugToJSONString())
		}
		if item.operation == witOpDel {
			wt, err = tree.DeleteWithWitness(*newHashFromStr(item.key))
			if err != nil {
				t.Fatal(err)
			}
			witnesses = append(witnesses, wt)

			t.Logf("Witness delete-%s: %s", item.key, wt.proof.DebugToString())

			t.Logf("Tree delete witness-%s", item.key)
			t.Logf("%s", tree.merkleTree.DebugToJSONString())
		}
	}

	ptree := NewPartialTree(rootHash, witnesses)
	for _, item := range items {
		if item.operation == witOpSet {
			ptree.Set(*newHashFromStr(item.key), hashNull)
		}
		if item.operation == witOpDel {
			ptree.Delete(*newHashFromStr(item.key))
		}
	}

	newRootHash := tree.Hash()
	newHash, _ := ptree.computeHash()
	if !newRootHash.IsEqual(&newHash) {
		t.Errorf("%s PartialTree hash is wrong. computed=%s expected=%s", name, newHash.String(), newRootHash.String())
	}
}

func TestStorageTree_general(t *testing.T) {
	doTestStorageTreeGeneral(
		t,
		"001",
		[]string{"01", "02", "03", "04", "05"},
		[]testItem{
			{witOpSet, "10"},
		},
	)

	doTestStorageTreeGeneral(
		t,
		"002",
		[]string{"05", "02", "04", "01", "03"},
		[]testItem{
			{witOpSet, "10"},
			{witOpSet, "11"},
			{witOpDel, "11"},
		},
	)

	doTestStorageTreeGeneral(
		t,
		"003",
		[]string{"05", "02", "04", "01", "03"},
		[]testItem{
			{witOpSet, "10"},
			{witOpSet, "11"},
			{witOpDel, "11"},
			{witOpDel, "01"},
			{witOpDel, "02"},
		},
	)

	doTestStorageTreeGeneral(
		t,
		"004",
		[]string{"02", "04"},
		[]testItem{
			{witOpSet, "10"},
			{witOpDel, "02"},
			{witOpDel, "10"},
		},
	)

	doTestStorageTreeGeneral(
		t,
		"005",
		[]string{"02", "04", "05", "01", "03"},
		[]testItem{
			{witOpSet, "10"},
			{witOpSet, "11"},
			{witOpDel, "02"},
			{witOpDel, "11"},
			{witOpDel, "01"},
		},
	)

	doTestStorageTreeGeneral(
		t,
		"005",
		[]string{"02", "04", "12", "13", "05", "01", "03", "06", "07", "08", "09", "10", "11", "14", "15"},
		[]testItem{
			{witOpSet, "20"},
			{witOpSet, "21"},
			{witOpDel, "02"},
			{witOpDel, "11"},
			{witOpDel, "01"},
			{witOpDel, "15"},
			{witOpDel, "14"},
			{witOpSet, "30"},
		},
	)
}

func TestStorageTree_1(t *testing.T) {
}
