package serialization_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/phoreproject/synapse/serialization"
)

func TestEncodeDecode(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub := btcec.PublicKey(priv.PublicKey)
	addr := serialization.NewAddress(&pub)
	addr2, err := serialization.DecodeAddress(addr.ToString())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(addr2[:], addr[:]) {
		t.Fatal("deserialized address does not match serialized address")
	}
}
