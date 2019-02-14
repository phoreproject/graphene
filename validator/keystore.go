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
type FakeKeyStore struct {
	proposerSlots map[uint32]uint64
}

// NewFakeKeyStore creates a new fake key store.
func NewFakeKeyStore() FakeKeyStore {
	return FakeKeyStore{proposerSlots: make(map[uint32]uint64)}
}

// GetKeyForValidator gets the private key for the given validator ID.
func (f FakeKeyStore) GetKeyForValidator(v uint32) *bls.SecretKey {
	r := newXORShift(uint64(v + 1000))
	s, _ := bls.RandSecretKey(r)
	return s
}

// IncrementProposerSlots increments proposal slots when chosen to propose a block.
func (f FakeKeyStore) IncrementProposerSlots(v uint32) {
	if _, found := f.proposerSlots[v]; !found {
		f.proposerSlots[v] = 0
	} else {
		f.proposerSlots[v]++
	}
}

// GetProposerSlots gets the number of times this validator was chosen as a proposer since genesis.
func (f FakeKeyStore) GetProposerSlots(v uint32) uint64 {
	return f.proposerSlots[v]
}

// Keystore is an interface for retrieving keys from a keystore.
type Keystore interface {
	GetKeyForValidator(uint32) *bls.SecretKey
	IncrementProposerSlots(uint32)
	GetProposerSlots(uint32) uint64
}
