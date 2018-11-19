package bls

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Signature used in the BLS signature scheme.
type Signature struct{}

// Serialize gets the binary representation of the
// signature.
func (s Signature) Serialize() []byte {
	return []byte{}
}

// DeserializeSignature deserializes a binary signature
// into the actual signature.
func DeserializeSignature([]byte) (Signature, error) {
	return Signature{}, nil
}

// SecretKey used in the BLS scheme.
type SecretKey struct{}

// DerivePublicKey derives a public key from a secret key.
func (s SecretKey) DerivePublicKey() *PublicKey {
	return &PublicKey{}
}

// PublicKey corresponding to secret key used in the BLS scheme.
type PublicKey struct{}

// Copy returns a copy of the public key
func (p PublicKey) Copy() PublicKey {
	return p
}

// Hash gets the hash of a pubkey
func (p PublicKey) Hash() []byte {
	// TODO: placeholder
	return chainhash.HashB([]byte("test"))
}

// Sign a message using a secret key - in a beacon/validator client,
// this key will come from and be unlocked from the account keystore.
func Sign(sec *SecretKey, msg []byte) (*Signature, error) {
	return &Signature{}, nil
}

// VerifySig against a public key.
func VerifySig(pub *PublicKey, msg []byte, sig *Signature) (bool, error) {
	return true, nil
}

// AggregateSigs puts multiple signatures into one using the underlying
// BLS sum functions.
func AggregateSigs(sigs []*Signature) (*Signature, error) {
	return &Signature{}, nil
}

// AggregatePubKeys by adding them up
func AggregatePubKeys(pubs []*PublicKey) (*PublicKey, error) {
	return &PublicKey{}, nil
}
