package keystore

import (
	"encoding/hex"
	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/shard/transfer"
	"github.com/phoreproject/synapse/wallet/address"
)

// Keypair is a pair of private and public keys.
type Keypair struct {
	key secp256k1.PrivateKey
	pubkey secp256k1.PublicKey
}

// Transfer creates a transfer transaction to the given address for a given amount.
func (k *Keypair) Transfer(shardID uint32, nonce uint32, to address.Address, amount uint64) (*transfer.ShardTransaction, error) {
	fromPkhBytes := address.HashPubkey(&k.pubkey, shardID)
	var fromPkh [20]byte
	copy(fromPkh[:], fromPkhBytes)

	toPkh, err := to.ToPubkeyHash()
	if err != nil {
		return nil, err
	}

	tx := transfer.ShardTransaction{
		FromPubkeyHash: fromPkh,
		ToPubkeyHash:   toPkh,
		Amount:         amount,
		Nonce:          nonce,
	}

	message := tx.GetTransactionData()

	messageHash := chainhash.HashH(message[:])

	sigBytes, err := secp256k1.SignCompact(&k.key, messageHash[:], false)
	if err != nil {
		return nil, err
	}

	copy(tx.Signature[:], sigBytes)

	return &tx, nil
}

// GetAddress gets the address of the keypair.
func (k *Keypair) GetAddress(shardID uint32) address.Address {
	addr := address.PubkeyToAddress(&k.pubkey, shardID)
	return addr
}

// GetPubkeyHash gets the pubkey hash of this keypair for a certain shard.
func (k *Keypair) GetPubkeyHash(shardID uint32) [20]byte {
	var pkh [20]byte
	out := address.HashPubkey(&k.pubkey, shardID)
	copy(pkh[:], out)
	return pkh
}

// GenerateRandomKeypair generates a random keypair.
func GenerateRandomKeypair() (*Keypair, error) {
	randKey, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	pub := randKey.PubKey()

	return &Keypair{
		key: *randKey,
		pubkey: *pub,
	}, nil
}

// KeypairFromHex is a keypair from a hex string.
func KeypairFromHex(hexString string) (*Keypair, error) {
	privBytes, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}

	return KeypairFromBytes(privBytes), nil
}

// KeypairFromBytes is a keypair from a byte array.
func KeypairFromBytes(privBytes []byte) (*Keypair) {
	key, pub := secp256k1.PrivKeyFromBytes(privBytes)

	return &Keypair{
		key: *key,
		pubkey: *pub,
	}
}