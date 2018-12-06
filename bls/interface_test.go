package bls_test

import (
	"testing"

	"github.com/phoreproject/synapse/bls"
)

func TestBasicSignature(t *testing.T) {
	r := NewXORShift(1)

	s, _ := bls.RandSecretKey(r)

	p := s.DerivePublicKey()

	msg := []byte("test!")

	sig, err := bls.Sign(s, msg)
	if err != nil {
		t.Fatal(err)
	}

	valid, err := bls.VerifySig(p, msg, sig)
	if err != nil {
		t.Fatal(err)
	}

	if !valid {
		t.Fatal("signature is not valid and should be")
	}
}

type XORShift struct {
	state uint64
}

func NewXORShift(state uint64) *XORShift {
	return &XORShift{state}
}

func (xor *XORShift) Read(b []byte) (int, error) {
	for i := range b {
		x := xor.state
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		b[i] = uint8(x)
		xor.state = x
	}
	return len(b), nil
}

func TestAggregateSignatures(t *testing.T) {
	r := NewXORShift(1)

	s0, nil := bls.RandSecretKey(r)
	s1, nil := bls.RandSecretKey(r)
	s2, nil := bls.RandSecretKey(r)

	p0 := s0.DerivePublicKey()
	p1 := s1.DerivePublicKey()
	p2 := s2.DerivePublicKey()

	msg := []byte("test!")

	sig0, err := bls.Sign(s0, msg)
	if err != nil {
		t.Fatal(err)
	}
	sig1, err := bls.Sign(s1, msg)
	if err != nil {
		t.Fatal(err)
	}
	sig2, err := bls.Sign(s2, msg)
	if err != nil {
		t.Fatal(err)
	}

	aggregateSig, err := bls.AggregateSigs([]*bls.Signature{sig0, sig1, sig2})
	if err != nil {
		t.Fatal(err)
	}

	valid := bls.VerifyAggregateCommon([]*bls.PublicKey{p0, p1, p2}, msg, aggregateSig)
	if !valid {
		t.Fatal("aggregate signature was not valid")
	}
}

func TestSerializeDeserializeSignature(t *testing.T) {
	r := NewXORShift(1)

	k, _ := bls.RandSecretKey(r)

	sig, err := bls.Sign(k, []byte("testing!"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = bls.DeserializeSignature(sig.Serialize())
	if err != nil {
		t.Fatal(err)
	}

	// TODO: verify sig == sigAfter
}

func TestHashPubkey(t *testing.T) {
	r := NewXORShift(1)

	k, _ := bls.RandSecretKey(r)

	p := k.DerivePublicKey()

	p.Hash()

	// TODO: verify small change changes hash
}

func TestCopyPubkey(t *testing.T) {
	r := NewXORShift(1)

	k, _ := bls.RandSecretKey(r)

	p := k.DerivePublicKey()

	p.Copy()

	// TODO: verify change in original does not change copy
}
