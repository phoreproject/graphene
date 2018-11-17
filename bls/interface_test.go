package bls_test

import (
	"testing"

	"github.com/phoreproject/synapse/bls"
)

func TestBasicSignature(t *testing.T) {
	s := &bls.SecretKey{}

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

func TestAggregateSignatures(t *testing.T) {
	s0 := &bls.SecretKey{}
	s1 := &bls.SecretKey{}
	s2 := &bls.SecretKey{}

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

	aggregatePubKey, err := bls.AggregatePubKeys([]*bls.PublicKey{p0, p1, p2})
	if err != nil {
		t.Fatal(err)
	}

	valid, err := bls.VerifySig(aggregatePubKey, msg, aggregateSig)
	if err != nil {
		t.Fatal(err)
	}

	if !valid {
		t.Fatal("aggregate signature was not valid")
	}
}

func TestSerializeDeserializeSignature(t *testing.T) {
	k := bls.SecretKey{}

	sig, err := bls.Sign(&k, []byte("testing!"))
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
	k := bls.SecretKey{}

	p := k.DerivePublicKey()

	p.Hash()

	// TODO: verify small change changes hash
}

func TestCopyPubkey(t *testing.T) {
	k := bls.SecretKey{}

	p := k.DerivePublicKey()

	p.Copy()

	// TODO: verify change in original does not change copy
}
