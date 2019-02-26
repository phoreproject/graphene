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

// NewFakeKeyStore creates a new fake key store.
func NewFakeKeyStore() FakeKeyStore {
	return FakeKeyStore{}
}

// GetKeyForValidator gets the private key for the given validator ID.
func (f FakeKeyStore) GetKeyForValidator(v uint32) *bls.SecretKey {
	r := newXORShift(uint64(v + 1000))
	s, _ := bls.RandSecretKey(r)
	return s
}

// GetPublicKeyForValidator gets the public key for the given validator ID.
func (f FakeKeyStore) GetPublicKeyForValidator(v uint32) *bls.PublicKey {
	r := newXORShift(uint64(v + 1000))
	s, _ := bls.RandSecretKey(r)
	return s.DerivePublicKey()
}

// MemoryKeyStore is a keystore that keeps keys in memory.
type MemoryKeyStore struct {
	privateKeys map[uint32]*bls.SecretKey
	publicKeys  map[uint32]*bls.PublicKey
}

// NewMemoryKeyStore creates a new memory key store.
func NewMemoryKeyStore(keys map[uint32]*bls.SecretKey) *MemoryKeyStore {
	return &MemoryKeyStore{privateKeys: keys, publicKeys: make(map[uint32]*bls.PublicKey)}
}

// GetKeyForValidator gets the private key for the given validator ID.
func (m *MemoryKeyStore) GetKeyForValidator(v uint32) *bls.SecretKey {
	return m.privateKeys[v]
}

// GetPublicKeyForValidator gets the public key for the given validator ID.
func (m *MemoryKeyStore) GetPublicKeyForValidator(v uint32) *bls.PublicKey {
	if m.publicKeys[v] == nil {
		m.publicKeys[v] = m.privateKeys[v].DerivePublicKey()
	}
	return m.publicKeys[v]
}

// Keystore is an interface for retrieving keys from a keystore.
type Keystore interface {
	GetKeyForValidator(uint32) *bls.SecretKey
	GetPublicKeyForValidator(uint32) *bls.PublicKey
}
