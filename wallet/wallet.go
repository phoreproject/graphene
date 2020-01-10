package wallet

import (
	"context"
	"fmt"
	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/shard/execution"
	"github.com/phoreproject/synapse/shard/transfer"
	"github.com/phoreproject/synapse/wallet/address"
	"github.com/phoreproject/synapse/wallet/keystore"
	"google.golang.org/grpc"
)

// KeyInfo is information about the wallet key.
type KeyInfo struct {
	keypair      keystore.Keypair
	Address      address.Address
	CurrentNonce uint32
}

// Wallet is a wallet that keeps track of basic balances for addresses and allows sending/receiving of money.
type Wallet struct {
	RPCClient pb.RelayerRPCClient
	Keystore  map[address.Address]*KeyInfo
}

// NewWallet creates a new wallet using a shard connection.
func NewWallet(conn *grpc.ClientConn) Wallet {
	return Wallet{
		RPCClient: pb.NewRelayerRPCClient(conn),
		Keystore:  make(map[address.Address]*KeyInfo),
	}
}

// RedeemPremine redeems the premine to our address.
func (w *Wallet) RedeemPremine(to address.Address) error {
	toPkh, err := to.ToPubkeyHash()
	if err != nil {
		return err
	}

	_ = transfer.RedeemTransaction{
		ToPubkeyHash: toPkh,
	}

	//_, err = w.RPCClient.SubmitTransaction(context.Background(), &pb.ShardTransactionSubmission{
	//	ShardID: 0,
	//	Transaction: &pb.ShardTransaction{
	//		TransactionData: tx.Serialize(),
	//	},
	//})

	return err
}

// GetBalance gets the balance of an address.
func (w *Wallet) GetBalance(of address.Address) (uint64, error) {
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
func (w *Wallet) SendToAddress(from address.Address, to address.Address, amount uint64) error {
	key, found := w.Keystore[from]
	if !found {
		return fmt.Errorf("could not find key for address: %s", from)
	}

	_, err := key.keypair.Transfer(0, key.CurrentNonce, to, amount)
	if err != nil {
		return err
	}

	//_, err = w.RPCClient.SubmitTransaction(context.Background(), &pb.ShardTransactionSubmission{
	//	ShardID: 0,
	//	Transaction: &pb.ShardTransaction{
	//		TransactionData: st.Serialize(),
	//	},
	//})

	//if err != nil {
	//	return err
	//}

	w.Keystore[from].CurrentNonce++

	return nil
}

// GetNewAddress gets a new address and adds it to the keystore.
func (w *Wallet) GetNewAddress(shardID uint32) (address.Address, error) {
	priv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return "", err
	}

	pub := priv.PubKey()

	addr := address.PubkeyToAddress(pub, shardID)

	w.Keystore[addr] = &KeyInfo{

		Address:      addr,
		CurrentNonce: 0,
	}

	return addr, nil
}

// ImportPrivKey imports a private key.
func (w *Wallet) ImportPrivKey(privBytes []byte, shardID uint32) address.Address {
	pair := keystore.KeypairFromBytes(privBytes)
	addr := pair.GetAddress(shardID)

	w.Keystore[addr] = &KeyInfo{
		keypair:      *pair,
		Address:      addr,
		CurrentNonce: 0,
	}

	return addr
}
