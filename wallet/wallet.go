package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/shard/execution"
	"github.com/phoreproject/synapse/shard/transfer"
	"google.golang.org/grpc"
)

// KeyInfo is information about the wallet key.
type KeyInfo struct {
	PrivateKey   secp256k1.PrivateKey
	PublicKey    secp256k1.PublicKey
	Address      Address
	CurrentNonce uint32
}

// Wallet is a wallet that keeps track of basic balances for addresses and allows sending/receiving of money.
type Wallet struct {
	RPCClient pb.ShardRPCClient
	Keystore  map[Address]*KeyInfo
}

// NewWallet creates a new wallet using a shard connection.
func NewWallet(conn *grpc.ClientConn) Wallet {
	return Wallet{
		RPCClient: pb.NewShardRPCClient(conn),
		Keystore:  make(map[Address]*KeyInfo),
	}
}

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

// RedeemPremine redeems the premine to our address.
func (w *Wallet) RedeemPremine(to Address) error {
	toPkh, err := to.ToPubkeyHash()
	if err != nil {
		return err
	}

	tx := transfer.RedeemTransaction{
		ToPubkeyHash: toPkh,
	}

	_, err = w.RPCClient.SubmitTransaction(context.Background(), &pb.ShardTransactionSubmission{
		ShardID: 0,
		Transaction: &pb.ShardTransaction{
			TransactionData: tx.Serialize(),
		},
	})

	return err
}

// GetBalance gets the balance of an address.
func (w *Wallet) GetBalance(of Address) (uint64, error) {
	pkh, err := of.ToPubkeyHash()
	if err != nil {
		return 0, err
	}

	storageKey := execution.GetStorageHashForPath([]byte("balance"), pkh[:])

	bal, err := w.RPCClient.GetStateKey(context.Background(), &pb.GetStateKeyRequest{
		ShardID: 0,
		Key:     storageKey[:],
	})
	if err != nil {
		return 0, err
	}

	h, err := chainhash.NewHash(bal.Value)
	if err != nil {
		return 0, err
	}

	return execution.HashTo64(*h), nil
}

// SendToAddress sends money to the specified address from the specified address.
func (w *Wallet) SendToAddress(from Address, to Address, amount uint64) error {
	key, found := w.Keystore[from]
	if !found {
		return fmt.Errorf("could not find key for address: %s", from)
	}

	fromPkh, err := from.ToPubkeyHash()
	if err != nil {
		return err
	}

	toPkh, err := to.ToPubkeyHash()
	if err != nil {
		return err
	}

	tx := transfer.ShardTransaction{
		FromPubkeyHash: fromPkh,
		ToPubkeyHash:   toPkh,
		Amount:         amount,
		Nonce:          key.CurrentNonce,
	}

	message := tx.GetTransactionData()

	fmt.Printf("%x\n", message)

	messageHash := chainhash.HashH(message[:])

	sigBytes, err := secp256k1.SignCompact(&key.PrivateKey, messageHash[:], false)
	if err != nil {
		return err
	}

	copy(tx.Signature[:], sigBytes)

	_, err = w.RPCClient.SubmitTransaction(context.Background(), &pb.ShardTransactionSubmission{
		ShardID: 0,
		Transaction: &pb.ShardTransaction{
			TransactionData: tx.Serialize(),
		},
	})

	if err != nil {
		return err
	}

	w.Keystore[from].CurrentNonce++

	return nil
}

// GetNewAddress gets a new address and adds it to the keystore.
func (w *Wallet) GetNewAddress(shardID uint32) (Address, error) {
	priv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return "", err
	}

	pub := priv.PubKey()

	addr := PubkeyToAddress(pub, shardID)

	w.Keystore[addr] = &KeyInfo{
		PrivateKey:   *priv,
		PublicKey:    *pub,
		Address:      addr,
		CurrentNonce: 0,
	}

	return addr, nil
}

// ImportPrivKey imports a private key.
func (w *Wallet) ImportPrivKey(privBytes []byte, shardID uint32) Address {
	priv, pub := secp256k1.PrivKeyFromBytes(privBytes)

	addr := PubkeyToAddress(pub, shardID)

	w.Keystore[addr] = &KeyInfo{
		PrivateKey:   *priv,
		PublicKey:    *pub,
		Address:      addr,
		CurrentNonce: 0,
	}

	return addr
}
