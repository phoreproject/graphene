package shard

import (
	"testing"
)

func doTestStorageTreeSerializerGeneral(t *testing.T, name string, keys []string, items []testItem) {
	tree := buildStorageTree(keys)
	rootHash := tree.Hash()

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
		}
		if item.operation == witOpDel {
			wt, err = tree.DeleteWithWitness(*newHashFromStr(item.key))
			if err != nil {
				t.Fatal(err)
			}
			witnesses = append(witnesses, wt)
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

	serializer := NewPartialTreeSerializer(1)
	serializedData := serializer.Serialize(&ptree)
	newPtree, err := serializer.Deserialize(serializedData)
	if err != nil {
		t.Fatal(err)
	}

	newHash, _ := newPtree.computeHash()
	if !newRootHash.IsEqual(&newHash) {
		t.Errorf("%s PartialTree hash is wrong. computed=%s expected=%s", name, newHash.String(), newRootHash.String())
	}
}

func TestStorageTreeSerializer(t *testing.T) {
	doTestStorageTreeSerializerGeneral(
		t,
		"001",
		[]string{"01", "02", "03", "04", "05"},
		[]testItem{
			{witOpSet, "10"},
		},
	)

	doTestStorageTreeSerializerGeneral(
		t,
		"002",
		[]string{"05", "02", "04", "01", "03"},
		[]testItem{
			{witOpSet, "10"},
			{witOpSet, "11"},
			{witOpDel, "11"},
		},
	)

	doTestStorageTreeSerializerGeneral(
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

	doTestStorageTreeSerializerGeneral(
		t,
		"004",
		[]string{"02", "04"},
		[]testItem{
			{witOpSet, "10"},
			{witOpDel, "02"},
			{witOpDel, "10"},
		},
	)

	doTestStorageTreeSerializerGeneral(
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

	doTestStorageTreeSerializerGeneral(
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
