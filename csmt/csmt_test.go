package csmt

import (
	"testing"

	"github.com/phoreproject/synapse/chainhash"
)

func newHashFromStr(s string) *Hash {
	result, _ := chainhash.NewHashFromStr(s)
	return result
}

func nodeHashFunction(left *Hash, right *Hash) Hash {
	return chainhash.HashH(append(left[:], right[:]...))
}

func createCSMT() *CSMT {
	return &CSMT{
		root:             nil,
		nodeHashFunction: nodeHashFunction,
	}
}

func TestCSMTMembershipProof(t *testing.T) {
	csmt := createCSMT()

	var h *Hash

	h = newHashFromStr("fe")
	err := csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("ff")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("01")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("02")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("7f")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("03")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("04")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("05")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("06")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("08")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}
}

func TestCSMTNonMembershipProof(t *testing.T) {
	csmt := createCSMT()

	err := csmt.Insert(newHashFromStr("fe"), nil)
	if err != nil {
		t.Fatal(err)
	}
	err = csmt.Insert(newHashFromStr("09"), nil)
	if err != nil {
		t.Fatal(err)
	}
	err = csmt.Insert(newHashFromStr("01"), nil)
	if err != nil {
		t.Fatal(err)
	}
	err = csmt.Insert(newHashFromStr("f6"), nil)
	if err != nil {
		t.Fatal(err)
	}
	err = csmt.Insert(newHashFromStr("08"), nil)
	if err != nil {
		t.Fatal(err)
	}

	if csmt.GetProof(newHashFromStr("70")).IsMembershipProof() {
		t.Errorf("Key should not exist in CSMT")
	}
	if csmt.GetProof(newHashFromStr("71")).IsMembershipProof() {
		t.Errorf("Key should not exist in CSMT")
	}
	if csmt.GetProof(newHashFromStr("85")).IsMembershipProof() {
		t.Errorf("Key should not exist in CSMT")
	}
}

func TestCSMTDuplicatedHash(t *testing.T) {
	csmt := createCSMT()

	var h *Hash

	h = newHashFromStr("fe")
	err := csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if csmt.Insert(h, nil) == nil {
		t.Errorf("There should be error of duplicated keys")
	}

	h = newHashFromStr("77")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if csmt.Insert(h, nil) == nil {
		t.Errorf("There should be error of duplicated keys")
	}

	h = newHashFromStr("56")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if csmt.Insert(h, nil) == nil {
		t.Errorf("There should be error of duplicated keys")
	}

	h = newHashFromStr("ff")
	err = csmt.Insert(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	if csmt.Insert(h, nil) == nil {
		t.Errorf("There should be error of duplicated keys")
	}
}

func TestDistance(t *testing.T) {
	var dist int

	dist = distance(newHashFromStr("ff"), newHashFromStr("00"))
	if dist != 256 {
		t.Errorf("distance: got %d, expected %d", dist, 256)
	}

	dist = distance(newHashFromStr("7f"), newHashFromStr("00"))
	if dist != 255 {
		t.Errorf("distance: got %d, expected %d", dist, 255)
	}

	dist = distance(newHashFromStr("ffff"), newHashFromStr("00ff"))
	if dist != 248 {
		t.Errorf("distance: got %d, expected %d", dist, 248)
	}

	dist = distance(newHashFromStr("7fff"), newHashFromStr("00ff"))
	if dist != 247 {
		t.Errorf("distance: got %d, expected %d", dist, 247)
	}
}

func TestCSMTRemoveKey(t *testing.T) {
	insertKeys := []string{
		"03",
		"06",
		"02",
		"04",
		"01",
		"05",
	}
	removeKeys := []string{
		"04",
		"03",
		"01",
		"02",
		"05",
		"06",
	}

	csmt := createCSMT()

	for _, key := range insertKeys {
		h := newHashFromStr(key)
		csmt.Insert(h, nil)
		if !csmt.GetProof(h).IsMembershipProof() {
			t.Errorf("Key should exist in the tree")
		}
	}

	for _, key := range removeKeys {
		h := newHashFromStr(key)

		if !csmt.GetProof(h).IsMembershipProof() {
			t.Errorf("Key should exist in the tree")
		}

		err := csmt.Remove(h)
		if err != nil {
			t.Fatal(err)
		}
		if csmt.GetProof(h).IsMembershipProof() {
			t.Errorf("Key should not exist in the tree")
		}

		// remove non-existing key
		if csmt.Remove(h) == nil {
			t.Errorf("Double remove should return error")
		}
	}
}

func TestCSMTGetValue(t *testing.T) {
	insertKeys := []string{
		"03",
		"06",
		"02",
		"04",
		"01",
		"05",
	}

	insertValues := []int{
		3,
		6,
		2,
		4,
		1,
		5,
	}

	csmt := createCSMT()

	for i, key := range insertKeys {
		h := newHashFromStr(key)
		csmt.Insert(h, insertValues[i])
		if !csmt.GetProof(h).IsMembershipProof() {
			t.Errorf("Key should exist in the tree")
		}
	}
	for i, key := range insertKeys {
		h := newHashFromStr(key)
		v, err := csmt.GetValue(h)
		if err != nil {
			t.Fatal(err)
		}
		if v.(int) != insertValues[i] {
			t.Errorf("Wrong value")
		}
	}

}
