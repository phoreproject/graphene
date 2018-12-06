package validator

import (
	"github.com/phoreproject/synapse/bls"
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

// FakeKeyStore should be assumed to be insecure.
type FakeKeyStore struct{}

// GetKeyForValidator gets the private key for the given validator ID.
func (f *FakeKeyStore) GetKeyForValidator(v uint32) *bls.SecretKey {
	r := newXORShift(uint64(v))
	s, _ := bls.RandSecretKey(r)
	return s
}

// Keystore is an interface for retrieving keys from a keystore.
type Keystore interface {
	GetKeyForValidator(uint32) *bls.SecretKey
}
