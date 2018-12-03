package csmt

import (
	"testing"

	"crypto/sha256"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func newHashFromStr(s string) *Hash {
	result, _ := chainhash.NewHashFromStr(s)
	return result
}

func getHash(data []uint8) chainhash.Hash {
	var hash chainhash.Hash
	sha := sha256.New()
	hash.SetBytes(sha.Sum(data))
	return hash
}

func nodeHashFunction(left *Hash, right *Hash) Hash {
	return getHash(append(left[:], right[:]...))
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
	csmt.Insert(h)
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("ff")
	csmt.Insert(h)
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("01")
	csmt.Insert(h)
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("02")
	csmt.Insert(h)
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("7f")
	csmt.Insert(h)
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("03")
	csmt.Insert(h)
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("04")
	csmt.Insert(h)
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("05")
	csmt.Insert(h)
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("06")
	csmt.Insert(h)
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}

	h = newHashFromStr("08")
	csmt.Insert(h)
	if !csmt.GetProof(h).IsMembershipProof() {
		t.Errorf("Key should exist in CSMT")
	}
	if !csmt.GetProof(h).(MembershipProof).VerifyHashInTree(h, csmt) {
		t.Errorf("Key doesn't verify in CSMT")
	}
}

func TestCSMTNonMembershipProof(t *testing.T) {
	csmt := createCSMT()

	csmt.Insert(newHashFromStr("fe"))
	csmt.Insert(newHashFromStr("09"))
	csmt.Insert(newHashFromStr("01"))
	csmt.Insert(newHashFromStr("f6"))
	csmt.Insert(newHashFromStr("08"))

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
	csmt.Insert(h)
	if csmt.Insert(h) == nil {
		t.Errorf("There should be error of duplicated keys")
	}

	h = newHashFromStr("77")
	csmt.Insert(h)
	if csmt.Insert(h) == nil {
		t.Errorf("There should be error of duplicated keys")
	}

	h = newHashFromStr("56")
	csmt.Insert(h)
	if csmt.Insert(h) == nil {
		t.Errorf("There should be error of duplicated keys")
	}

	h = newHashFromStr("ff")
	csmt.Insert(h)
	if csmt.Insert(h) == nil {
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
