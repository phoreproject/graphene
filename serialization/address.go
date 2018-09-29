package serialization

import (
	"crypto/sha256"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/golangcrypto/ripemd160"

	"github.com/btcsuite/btcutil/base58"
)

// AddressVersion is the version of the Phore address.
const AddressVersion = byte(55)

// Address represents a public key hash address.
type Address [20]byte

// ToString gets a string from an address.
func (a Address) ToString() string {
	return base58.CheckEncode(a[:], byte(AddressVersion))
}

// DecodeAddress decodes an address into a string.
func DecodeAddress(addr string) (*Address, error) {
	var out Address
	res, ver, err := base58.CheckDecode(addr)
	if ver != AddressVersion {
		return nil, errors.New("address is not Phore address")
	}
	if err != nil {
		return nil, err
	}
	if len(res) != 20 {
		return nil, errors.New("address is not in pubkey-hash format")
	}
	copy(out[:], res)
	return &out, nil
}

// Hash160 calculates the Hash160 of the given bytes.
func Hash160(in []byte) []byte {
	s := sha256.New()
	h := ripemd160.New()
	return h.Sum(s.Sum(in))
}

// NewAddress gets an address from a public key.
func NewAddress(pubkey *btcec.PublicKey) *Address {
	compressedPubKey := pubkey.SerializeCompressed()
	h := Hash160(compressedPubKey)
	var out Address
	copy(out[:], h)
	return &out
}
