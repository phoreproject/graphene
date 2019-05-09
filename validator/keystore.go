package validator

import (
	"fmt"
	"sync"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
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

// HDReader is a random reader from a hash.
type HDReader struct {
	state chainhash.Hash
}

func (r *HDReader) Read(p []byte) (n int, err error) {
	length := len(p)
	sent := 0
	for sent < length {
		amountToSend := length - sent
		if amountToSend > chainhash.HashSize {
			amountToSend = chainhash.HashSize
		}
		copy(p[sent:sent+amountToSend], r.state[:])
		r.state = chainhash.HashH(r.state[:])
		sent += amountToSend
	}

	return sent, nil
}

// GetReaderForID gets a reader for a specific root key and validator ID.
func GetReaderForID(rootKey string, validatorID uint32) *HDReader {
	return &HDReader{state: chainhash.HashH([]byte(fmt.Sprintf("%s %d", rootKey, validatorID)))}
}

// RootKeyStore is a keystore where each validator is derived from a specific
// root key.
type RootKeyStore struct {
	rootKey string

	cachedPriv     map[uint32]*bls.SecretKey
	cachedPrivLock *sync.RWMutex

	cachedPub     map[uint32]*bls.PublicKey
	cachedPubLock *sync.RWMutex
}

// NewRootKeyStore gets a new root key store from the root key.
func NewRootKeyStore(rootKey string) *RootKeyStore {
	return &RootKeyStore{
		rootKey:        rootKey,
		cachedPriv:     make(map[uint32]*bls.SecretKey),
		cachedPrivLock: new(sync.RWMutex),
		cachedPub:      make(map[uint32]*bls.PublicKey),
		cachedPubLock:  new(sync.RWMutex),
	}
}

// GetKeyForValidator gets a private key for a validator ID.
func (k *RootKeyStore) GetKeyForValidator(validatorID uint32) *bls.SecretKey {
	k.cachedPrivLock.RLock()
	if key, found := k.cachedPriv[validatorID]; found {
		k.cachedPrivLock.RUnlock()
		return key
	}
	k.cachedPrivLock.RUnlock()
	k.cachedPrivLock.Lock()
	defer k.cachedPrivLock.Unlock()
	k.cachedPriv[validatorID], _ = bls.RandSecretKey(GetReaderForID(k.rootKey, validatorID))
	return k.cachedPriv[validatorID]
}

// GetPublicKeyForValidator gets a public key for a validator ID.
func (k *RootKeyStore) GetPublicKeyForValidator(validatorID uint32) *bls.PublicKey {
	// check if we have a cached public key
	k.cachedPubLock.RLock()
	if key, found := k.cachedPub[validatorID]; found {
		k.cachedPubLock.RUnlock()
		return key
	}
	k.cachedPubLock.RUnlock()

	// check if we have a cached private key
	k.cachedPrivLock.RLock()
	if key, found := k.cachedPriv[validatorID]; found {
		k.cachedPrivLock.RUnlock()
		pub := key.DerivePublicKey()
		k.cachedPubLock.Lock()
		defer k.cachedPubLock.Unlock()
		k.cachedPub[validatorID] = pub
		return pub
	}
	k.cachedPrivLock.RUnlock()

	// derive both private and public keys
	k.cachedPrivLock.Lock()
	defer k.cachedPrivLock.Unlock()
	k.cachedPubLock.Lock()
	defer k.cachedPubLock.Unlock()
	k.cachedPriv[validatorID], _ = bls.RandSecretKey(GetReaderForID(k.rootKey, validatorID))
	k.cachedPub[validatorID] = k.cachedPriv[validatorID].DerivePublicKey()
	return k.cachedPub[validatorID]
}

// Keystore is an interface for retrieving keys from a keystore.
type Keystore interface {
	GetKeyForValidator(uint32) *bls.SecretKey
	GetPublicKeyForValidator(uint32) *bls.PublicKey
}
