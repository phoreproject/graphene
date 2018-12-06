package serialization_test

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcutil/base58"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/serialization"
)

type xorshift struct {
	state uint64
}

func newXORShift(state uint64) *xorshift {
	return &xorshift{state}
}

func (xor *xorshift) Read(b []byte) (int, error) {
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

func TestEncodeDecode(t *testing.T) {
	x := newXORShift(29)
	key, _ := bls.RandSecretKey(x)
	pub := key.DerivePublicKey()
	addr := serialization.NewAddress(pub)
	addr2, err := serialization.DecodeAddress(addr.ToString())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(addr2[:], addr[:]) {
		t.Fatal("deserialized address does not match serialized address")
	}
}

func TestDecodeInvalidAddresses(t *testing.T) {
	validAddr := "PP8g6YkR881QEzYqTzmYoCSHFrwHe6htJV"

	invalidPrefix := base58.CheckEncode([]byte("aaaaaaaaaaaaaaaaaaaa"), serialization.AddressVersion+1)

	_, err := serialization.DecodeAddress(invalidPrefix)
	if err == nil {
		t.Fatal("invalid prefix was accepted by decode address")
	}

	invalidChecksum := validAddr[:32] + "ht"

	_, err = serialization.DecodeAddress(invalidChecksum)
	if err == nil {
		t.Fatal("invalid checksum was accepted by decode address")
	}

	invalidLength := base58.CheckEncode([]byte("aaaaaaaaaa"), serialization.AddressVersion)
	_, err = serialization.DecodeAddress(invalidLength)
	if err == nil {
		t.Fatal("invalid length was accepted by decode address")
	}
}
