package validator

import (
	"github.com/phoreproject/synapse/bls"
)

// FakeKeyStore should be assumed to be insecure.
type FakeKeyStore struct{}

// GetKeyForValidator gets the private key for the given validator ID.
func (f *FakeKeyStore) GetKeyForValidator(v uint32) *bls.SecretKey {
	return &bls.SecretKey{}
}

// Keystore is an interface for retrieving keys from a keystore.
type Keystore interface {
	GetKeyForValidator(uint32) *bls.SecretKey
}
