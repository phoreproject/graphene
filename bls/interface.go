package bls

import (
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/bls"
)

// Signature used in the BLS signature scheme.
type Signature struct {
	s bls.Signature
}

// Serialize gets the binary representation of the
// signature.
func (s Signature) Serialize() []byte {
	return s.s.Serialize()
}

// DeserializeSignature deserializes a binary signature
// into the actual signature.
func DeserializeSignature(b []byte) (*Signature, error) {
	s, err := bls.DeserializeSignature(b)
	if err != nil {
		return nil, err
	}

	return &Signature{s: *s}, nil
}

// SecretKey used in the BLS scheme.
type SecretKey struct {
	s bls.SecretKey
}

// RandSecretKey generates a random key given a byte reader.
func RandSecretKey(r io.Reader) (*SecretKey, error) {
	key, err := bls.RandKey(r)
	if err != nil {
		return nil, err
	}

	return &SecretKey{s: *key}, nil
}

// DerivePublicKey derives a public key from a secret key.
func (s SecretKey) DerivePublicKey() *PublicKey {
	pub := bls.PrivToPub(&s.s)
	return &PublicKey{p: *pub}
}

func (p PublicKey) String() string {
	return p.p.String()
}

// PublicKey corresponding to secret key used in the BLS scheme.
type PublicKey struct {
	p bls.PublicKey
}

// Copy returns a copy of the public key
func (p PublicKey) Copy() PublicKey {
	return p
}

// Hash gets the hash of a pubkey
func (p PublicKey) Hash() []byte {
	return chainhash.HashB(p.p.Serialize())
}

// Serialize serializes a pubkey to bytes
func (p PublicKey) Serialize() []byte {
	return p.p.Serialize()
}

// DeserializePubKey deserializes a public key from bytes.
func DeserializePubKey(b []byte) (*PublicKey, error) {
	pub, err := bls.DeserializePublicKey(b)
	if err != nil {
		return nil, err
	}

	return &PublicKey{p: *pub}, nil
}

// Sign a message using a secret key - in a beacon/validator client,
// this key will come from and be unlocked from the account keystore.
func Sign(sec *SecretKey, msg []byte) (*Signature, error) {
	s := bls.Sign(msg, &sec.s)
	return &Signature{s: *s}, nil
}

// VerifySig against a public key.
func VerifySig(pub *PublicKey, msg []byte, sig *Signature) (bool, error) {
	return bls.Verify(msg, &pub.p, &sig.s), nil
}

// AggregateSigs puts multiple signatures into one using the underlying
// BLS sum functions.
func AggregateSigs(sigs []*Signature) (*Signature, error) {
	blsSigs := make([]*bls.Signature, len(sigs))
	for i := range sigs {
		blsSigs[i] = &sigs[i].s
	}
	aggSig := bls.AggregateSignatures(blsSigs)
	return &Signature{s: *aggSig}, nil
}

// VerifyAggregate verifies a signature over many messages.
func VerifyAggregate(pubkeys []*PublicKey, msgs [][]byte, signature *Signature) bool {
	if len(pubkeys) != len(msgs) {
		return false
	}

	blsPubs := make([]*bls.PublicKey, len(pubkeys))
	for i := range pubkeys {
		blsPubs[i] = &pubkeys[i].p
	}

	return signature.s.VerifyAggregate(blsPubs, msgs)
}

// VerifyAggregateCommon verifies a signature over a common message.
func VerifyAggregateCommon(pubkeys []*PublicKey, msg []byte, signature *Signature) bool {
	blsPubs := make([]*bls.PublicKey, len(pubkeys))
	for i := range pubkeys {
		blsPubs[i] = &pubkeys[i].p
	}

	return signature.s.VerifyAggregateCommon(blsPubs, msg)
}

// AggregatePubKeys aggregates some public keys into one.
func AggregatePubKeys(pubkeys []*PublicKey) *PublicKey {
	blsPubs := make([]*bls.PublicKey, len(pubkeys))
	for i := range pubkeys {
		blsPubs[i] = &pubkeys[i].p
	}

	newPub := bls.AggregatePublicKeys(blsPubs)

	return &PublicKey{p: *newPub}
}

// AggregatePubKey adds another public key to this one.
func (p *PublicKey) AggregatePubKey(other *PublicKey) {
	p.p.Aggregate(&other.p)
}

// AggregateSig adds another signature to this one.
func (s *Signature) AggregateSig(other *Signature) {
	s.s.Aggregate(&other.s)
}

// NewAggregatePublicKey creates a blank public key.
func NewAggregatePublicKey() *PublicKey {
	pub := bls.NewAggregatePubkey()
	return &PublicKey{p: *pub}
}

// NewAggregateSignature creates a blank signature key.
func NewAggregateSignature() *Signature {
	sig := bls.NewAggregateSignature()
	return &Signature{s: *sig}
}
