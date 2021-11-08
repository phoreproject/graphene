package bls_test

import (
	"bytes"
	"testing"

	"github.com/phoreproject/graphene/bls"
	"github.com/phoreproject/graphene/chainhash"
)

func TestBasicSignature(t *testing.T) {
	r := NewXORShift(1)

	s, _ := bls.RandSecretKey(r)

	p := s.DerivePublicKey()

	msg := []byte("test!")

	sig, err := bls.Sign(s, msg, 0)
	if err != nil {
		t.Fatal(err)
	}

	valid, err := bls.VerifySig(p, msg, sig, 0)
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

	s0, _ := bls.RandSecretKey(r)
	s1, _ := bls.RandSecretKey(r)
	s2, _ := bls.RandSecretKey(r)

	p0 := s0.DerivePublicKey()
	p1 := s1.DerivePublicKey()
	p2 := s2.DerivePublicKey()

	msg := []byte("test!")

	sig0, err := bls.Sign(s0, msg, 0)
	if err != nil {
		t.Fatal(err)
	}
	sig1, err := bls.Sign(s1, msg, 0)
	if err != nil {
		t.Fatal(err)
	}
	sig2, err := bls.Sign(s2, msg, 0)
	if err != nil {
		t.Fatal(err)
	}

	aggregateSig, err := bls.AggregateSigs([]*bls.Signature{sig0, sig1, sig2})
	if err != nil {
		t.Fatal(err)
	}

	valid := bls.VerifyAggregateCommon([]*bls.PublicKey{p0, p1, p2}, msg, aggregateSig, 0)
	if !valid {
		t.Fatal("aggregate signature was not valid")
	}
}

func TestVerifyAggregate(t *testing.T) {
	r := NewXORShift(1)

	s0, _ := bls.RandSecretKey(r)
	s1, _ := bls.RandSecretKey(r)
	s2, _ := bls.RandSecretKey(r)

	p0 := s0.DerivePublicKey()
	p1 := s1.DerivePublicKey()
	p2 := s2.DerivePublicKey()

	msg0 := []byte("test!")
	msg1 := []byte("test! 1")
	msg2 := []byte("test! 2")

	sig0, err := bls.Sign(s0, msg0, 0)
	if err != nil {
		t.Fatal(err)
	}
	sig1, err := bls.Sign(s1, msg1, 0)
	if err != nil {
		t.Fatal(err)
	}
	sig2, err := bls.Sign(s2, msg2, 0)
	if err != nil {
		t.Fatal(err)
	}

	aggregateSig, err := bls.AggregateSigs([]*bls.Signature{sig0, sig1, sig2})
	if err != nil {
		t.Fatal(err)
	}

	valid := bls.VerifyAggregate([]*bls.PublicKey{p0, p1, p2}, [][]byte{msg0, msg1, msg2}, aggregateSig, 0)
	if !valid {
		t.Fatal("aggregate signature was not valid")
	}
}

func TestVerifyAggregateSeparate(t *testing.T) {
	r := NewXORShift(1)

	s0, _ := bls.RandSecretKey(r)
	s1, _ := bls.RandSecretKey(r)
	s2, _ := bls.RandSecretKey(r)

	p0 := s0.DerivePublicKey()
	p1 := s1.DerivePublicKey()
	p2 := s2.DerivePublicKey()

	msg0 := []byte("test!")

	sig0, err := bls.Sign(s0, msg0, 0)
	if err != nil {
		t.Fatal(err)
	}
	sig1, err := bls.Sign(s1, msg0, 0)
	if err != nil {
		t.Fatal(err)
	}
	sig2, err := bls.Sign(s2, msg0, 0)
	if err != nil {
		t.Fatal(err)
	}

	aggregateSig, err := bls.AggregateSigs([]*bls.Signature{sig0, sig1, sig2})
	if err != nil {
		t.Fatal(err)
	}

	aggPk := bls.NewAggregatePublicKey()
	aggPk.AggregatePubKey(p0)
	aggPk.AggregatePubKey(p1)
	aggPk.AggregatePubKey(p2)

	valid, err := bls.VerifySig(aggPk, msg0, aggregateSig, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("aggregate signature was not valid")
	}

	aggPk = bls.AggregatePubKeys([]*bls.PublicKey{p0, p1, p2})
	valid, err = bls.VerifySig(aggPk, msg0, aggregateSig, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("aggregate signature was not valid")
	}

	aggregateSig = bls.NewAggregateSignature()
	aggregateSig.AggregateSig(sig0)
	aggregateSig.AggregateSig(sig1)
	aggregateSig.AggregateSig(sig2)
	valid, err = bls.VerifySig(aggPk, msg0, aggregateSig, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("aggregate signature was not valid")
	}
}

func TestSerializeDeserializeSignature(t *testing.T) {
	r := NewXORShift(1)

	k, _ := bls.RandSecretKey(r)
	pub := k.DerivePublicKey()

	sig, err := bls.Sign(k, []byte("testing!"), 0)
	if err != nil {
		t.Fatal(err)
	}

	sigAfter, err := bls.DeserializeSignature(sig.Serialize())
	if err != nil {
		t.Fatal(err)
	}

	valid, err := bls.VerifySig(pub, []byte("testing!"), sigAfter, 0)
	if err != nil {
		t.Fatal(err)
	}

	if !valid {
		t.Fatal("signature did not verify")
	}
}

func TestSerializeDeserializeSecret(t *testing.T) {
	r := NewXORShift(1)

	k, _ := bls.RandSecretKey(r)
	pub := k.DerivePublicKey()

	kSer := k.Serialize()
	kNew := bls.DeserializeSecretKey(kSer)

	sig, err := bls.Sign(&kNew, chainhash.HashB([]byte("testing!")), 0)
	if err != nil {
		t.Fatal(err)
	}

	valid, err := bls.VerifySig(pub, chainhash.HashB([]byte("testing!")), sig, 0)
	if err != nil {
		t.Fatal(err)
	}

	if !valid {
		t.Fatal("signature did not verify")
	}
}

func TestCopyPubkey(t *testing.T) {
	r := NewXORShift(1)

	k, _ := bls.RandSecretKey(r)

	p := k.DerivePublicKey()
	p2 := p.Copy()

	p.Copy()

	p.AggregatePubKey(&p2)

	if p2.Equals(*p) {
		t.Fatal("pubkey copy is incorrect")
	}
}

func TestCopySignature(t *testing.T) {
	r := NewXORShift(1)

	k, _ := bls.RandSecretKey(r)

	s, err := bls.Sign(k, []byte{}, 0)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := bls.Sign(k, []byte{}, 1)
	if err != nil {
		t.Fatal(err)
	}

	sCopy := s.Copy()

	s.AggregateSig(s2)
	sSer := s.Serialize()
	sCopySer := sCopy.Serialize()
	if bytes.Equal(sSer[:], sCopySer[:]) {
		t.Fatal("copy returns pointer")
	}
}
