package mempool_test

import (
	"bytes"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/relayer/mempool"
	"github.com/phoreproject/synapse/shard/execution"
	"github.com/phoreproject/synapse/shard/transfer"
	"github.com/phoreproject/synapse/wallet/keystore"
	"testing"
)


func RedeemPremine(key *keystore.Keypair, shardID uint32) []byte {
	tx := transfer.RedeemTransaction{
		ToPubkeyHash: key.GetPubkeyHash(shardID),
	}

	return tx.Serialize()
}

var premineKey, _ = keystore.KeypairFromHex("22a47fa09a223f2aa079edf85a7c2d4f8720ee63e502ee2869afab7de234b80c")

func TestShardMempoolAddNormal(t *testing.T) {
	stateDB := csmt.NewInMemoryTreeDB()
	shardInfo := execution.ShardInfo{
		CurrentCode: transfer.Code,
		ShardID: 0,
	}

	sm := mempool.NewShardMempool(stateDB, shardInfo)

	premineTx := RedeemPremine(premineKey, 0)

	err := sm.Add(premineTx)
	if err != nil {
		t.Fatal(err)
	}

	mempoolReturn, _, err := sm.GetTransactions(-1)
	if err != nil {
		t.Fatal(err)
	}

	if len(mempoolReturn) != 1 {
		t.Fatal("expected mempool to return one transaction")
	}

	if !bytes.Equal(mempoolReturn[0], premineTx) {
		t.Fatal("expected mempool to return redeem transaction")
	}
}


func TestShardMempoolAddInvalid(t *testing.T) {
	stateDB := csmt.NewInMemoryTreeDB()
	shardInfo := execution.ShardInfo{
		CurrentCode: transfer.Code,
		ShardID: 0,
	}

	sm := mempool.NewShardMempool(stateDB, shardInfo)

	premineTx := transfer.RedeemTransaction{
		ToPubkeyHash: [20]byte{},
	}

	err := sm.Add(premineTx.Serialize())
	if err == nil {
		t.Fatal("expected mempool to reject invalid transaction")
	}
}

func TestShardMempoolAddConflicting(t *testing.T) {
	stateDB := csmt.NewInMemoryTreeDB()
	shardInfo := execution.ShardInfo{
		CurrentCode: transfer.Code,
		ShardID: 0,
	}


	premineTx := RedeemPremine(premineKey, 0)

	stateTree := csmt.NewTree(stateDB)

	err := stateTree.Update(func (tx csmt.TreeTransaction) error {
		_, err := execution.Transition(tx, premineTx, shardInfo)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	sm := mempool.NewShardMempool(stateDB, shardInfo)

	to1, err := keystore.GenerateRandomKeypair()
	if err != nil {
		t.Fatal(err)
	}

	to2, err := keystore.GenerateRandomKeypair()
	if err != nil {
		t.Fatal(err)
	}

	tx1, err := premineKey.Transfer(0, 0, to1.GetAddress(0), 1)
	if err != nil {
		t.Fatal(err)
	}
	tx2, err := premineKey.Transfer(0, 0, to2.GetAddress(0), 1)
	if err != nil {
		t.Fatal(err)
	}
	tx1Ser := tx1.Serialize()
	err = sm.Add(tx1.Serialize())
	if err != nil {
		t.Fatal(err)
	}
	err = sm.Add(tx2.Serialize())
	if err != nil {
		t.Fatal(err)
	}

	txs, _, err := sm.GetTransactions(-1)
	if err != nil {
		t.Fatal(err)
	}

	if len(txs) != 1 {
		t.Fatal("expected 1 transactions to be included")
	}

	if !bytes.Equal(tx1Ser, txs[0]) {
		t.Fatal("expected first transaction to match first transfer transaction")
	}
}