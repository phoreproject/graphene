package bls

import (
	"io"

	"github.com/phoreproject/bls"
	"github.com/phoreproject/synapse/chainhash"
)

const (
	// DomainProposal is a signature for proposing a block.
	DomainProposal = iota

	// DomainAttestation is a signature for an attestation.
	DomainAttestation

	// DomainDeposit is a signature for validating a deposit.
	DomainDeposit

	// DomainExit is a signature for a validator exit.
	DomainExit
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

// Copy returns a copy of the signature.
func (s Signature) Copy() *Signature {
	c := s.s.Copy()
	return &Signature{*c}
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

// EmptySignature is an empty signature.
var EmptySignature, _ = DeserializeSignature(make([]byte, 48))

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

// Serialize serializes a public key to bytes.
func (p PublicKey) Serialize() []byte {
	return p.p.Serialize()
}

// Equals checks if two public keys are equal.
func (p PublicKey) Equals(other PublicKey) bool {
	return p.p.Equals(other.p)
}

// DeserializePublicKey deserialies a public key from the provided bytes.
func DeserializePublicKey(b []byte) (*PublicKey, error) {
	p, err := bls.DeserializePublicKey(b)
	if err != nil {
		return nil, err
	}
	return &PublicKey{*p}, nil
}

// Copy returns a copy of the public key
func (p PublicKey) Copy() PublicKey {
	return p
}

// Hash gets the hash of a pubkey
func (p PublicKey) Hash() []byte {
	return chainhash.HashB(p.p.Serialize())
}

// EncodeSSZ implements Encodable
func (p PublicKey) EncodeSSZ(writer io.Writer) error {
	writer.Write(p.Serialize())

	return nil
}

// EncodeSSZSize implements Encodable
func (p PublicKey) EncodeSSZSize() (uint32, error) {
	return 96, nil // hard coded?
	//return (uint32)(len(p.Serialize())), nil
}

// DecodeSSZ implements Decodable
func (p PublicKey) DecodeSSZ(reader io.Reader) error {
	size, _ := p.EncodeSSZSize()
	buf := make([]byte, size)
	key, err := DeserializePublicKey(buf)
	if err != nil {
		return err
	}
	p.p = key.p
	return nil
}

// EncodeSSZ implements Encodable
func (s Signature) EncodeSSZ(writer io.Writer) error {
	writer.Write(s.Serialize())

	return nil
}

// EncodeSSZSize implements Encodable
func (s Signature) EncodeSSZSize() (uint32, error) {
	return 48, nil // hard coded?
	//return (uint32)(len(p.Serialize())), nil
}

// DecodeSSZ implements Decodable
func (s Signature) DecodeSSZ(reader io.Reader) error {
	size, _ := s.EncodeSSZSize()
	buf := make([]byte, size)
	sig, err := DeserializeSignature(buf)
	if err != nil {
		return err
	}
	s.s = sig.s
	return nil
}

// Sign a message using a secret key - in a beacon/validator client,
// this key will come from and be unlocked from the account keystore.
func Sign(sec *SecretKey, msg []byte, domain uint64) (*Signature, error) {
	s := bls.Sign(msg, &sec.s, domain)
	return &Signature{s: *s}, nil
}

// VerifySig against a public key.
func VerifySig(pub *PublicKey, msg []byte, sig *Signature, domain uint64) (bool, error) {
	return bls.Verify(msg, &pub.p, &sig.s, domain), nil
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
func VerifyAggregate(pubkeys []*PublicKey, msgs [][]byte, signature *Signature, domain uint64) bool {
	if len(pubkeys) != len(msgs) {
		return false
	}

	blsPubs := make([]*bls.PublicKey, len(pubkeys))
	for i := range pubkeys {
		blsPubs[i] = &pubkeys[i].p
	}

	return signature.s.VerifyAggregate(blsPubs, msgs, domain)
}

// VerifyAggregateCommon verifies a signature over a common message.
func VerifyAggregateCommon(pubkeys []*PublicKey, msg []byte, signature *Signature, domain uint64) bool {
	blsPubs := make([]*bls.PublicKey, len(pubkeys))
	for i := range pubkeys {
		blsPubs[i] = &pubkeys[i].p
	}

	return signature.s.VerifyAggregateCommon(blsPubs, msg, domain)
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
