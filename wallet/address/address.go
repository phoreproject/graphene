package address

import (
	"encoding/binary"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/phoreproject/synapse/chainhash"
)

// PhoreAddressVersion is the version used for addresses.
const PhoreAddressVersion = 50

// Address is a Phore address.
type Address string

// PubkeyToAddress converts a pubkey to an address.
func PubkeyToAddress(pubkey *secp256k1.PublicKey, shardID uint32) Address {
	var shardBytes [4]byte
	binary.BigEndian.PutUint32(shardBytes[:], shardID)

	pkh := chainhash.HashH(append(shardBytes[:], pubkey.SerializeCompressed()...))

	addr := base58.CheckEncode(pkh[:20], PhoreAddressVersion)

	return Address(addr)
}

// HashPubkey converts a public key into a pubkey hash.
func HashPubkey(pubkey *secp256k1.PublicKey, shardID uint32) []byte {
	var shardBytes [4]byte
	binary.BigEndian.PutUint32(shardBytes[:], shardID)

	pkh := chainhash.HashH(append(shardBytes[:], pubkey.SerializeCompressed()...))

	return pkh[:20]
}

// ToPubkeyHash converts an address to a pubkey hash.
func (a Address) ToPubkeyHash() ([20]byte, error) {
	pkh, version, err := base58.CheckDecode(string(a))
	if err != nil {
		return [20]byte{}, err
	}

	if version != PhoreAddressVersion {
		return [20]byte{}, fmt.Errorf("invalid version, expecting: %d, got: %d", PhoreAddressVersion, version)
	}

	if len(pkh) != 20 {
		return [20]byte{}, fmt.Errorf("invalid address length, expected: 20, go: %d", len(pkh))
	}

	var out [20]byte
	copy(out[:], pkh)
	return out, nil
}

// ValidateAddress returns true if the address passed is valid.
func ValidateAddress(a string) bool {
	pkh, version, err := base58.CheckDecode(string(a))
	if err != nil {
		return false
	}

	if version != PhoreAddressVersion {
		return false
	}

	if len(pkh) != 20 {
		return false
	}

	return true
}