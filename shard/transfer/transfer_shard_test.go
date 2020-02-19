package transfer

import (
	"encoding/hex"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/shard/state"
	"io/ioutil"
	"os"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1"

	"github.com/phoreproject/synapse/chainhash"
)

func TestTransferShard(t *testing.T) {
	shardFile, err := os.Open("transfer_shard.wasm")
	if err != nil {
		t.Fatal(err)
	}

	shardCode, err := ioutil.ReadAll(shardFile)
	if err != nil {
		t.Fatal(err)
	}

	treeDB := csmt.NewInMemoryTreeDB()
	tree := csmt.NewTree(treeDB)
	store := state.NewFullShardState(tree)

	pkBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2d4f87" +
		"20ee63e502ee2869afab7de234b80c")
	if err != nil {
		t.Fatal(err)
	}
	privkey, pubKey := secp256k1.PrivKeyFromBytes(pkBytes)

	var pubkeyFrom [33]byte
	copy(pubkeyFrom[:], pubKey.SerializeCompressed())

	var shardBytes [4]byte

	fromAddressHash := chainhash.HashH(append(shardBytes[:], pubkeyFrom[:]...))

	storageHashPubkeyFrom := state.GetStorageHashForPath([]byte("balance"), fromAddressHash[:20])

	err = store.Update(func(a state.AccessInterface) error {
		err = a.Set(storageHashPubkeyFrom, state.Uint64ToHash(100))
		if err != nil {
			t.Fatal(err)
		}

		s, err := state.NewShard(shardCode, []int64{8}, a, 0)
		if err != nil {
			t.Fatal(err)
		}

		var fromAddress [20]byte
		copy(fromAddress[:], fromAddressHash[:20])

		txBytes := ShardTransaction{
			FromPubkeyHash: fromAddress,
			ToPubkeyHash:   [20]byte{},
			Amount:         10,
			Nonce:          0,
		}

		message := txBytes.GetTransactionData()

		messageHash := chainhash.HashH(message[:])

		sigBytes, err := secp256k1.SignCompact(privkey, messageHash[:], false)
		if err != nil {
			t.Fatal(err)
		}

		copy(txBytes.Signature[:], sigBytes)

		txContext, err := state.LoadArgumentContextFromTransaction(txBytes.Serialize())
		if err != nil {
			t.Fatal(err)
		}

		code, err := s.RunFunc(txContext)
		if err != nil {
			t.Fatal(err)
		}

		if code.(uint64) != 0 {
			t.Fatalf("function exited with non-zero exit code: %d", code)
		}

		// next test sending the same transaction again (should fail due to used nonce)
		code, err = s.RunFunc(txContext)
		if err != nil {
			t.Fatal(err)
		}

		if code.(uint64) == 0 {
			t.Fatalf("function exited with zero exit code: %d", code)
		}

		// next test sending the same transaction with a different nonce
		txBytes = ShardTransaction{
			FromPubkeyHash: fromAddress,
			ToPubkeyHash:   [20]byte{},
			Amount:         10,
			Nonce:          1,
		}

		message = txBytes.GetTransactionData()

		messageHash = chainhash.HashH(message[:])

		sigBytes, err = secp256k1.SignCompact(privkey, messageHash[:], false)
		if err != nil {
			t.Fatal(err)
		}

		copy(txBytes.Signature[:], sigBytes)

		txContext, err = state.LoadArgumentContextFromTransaction(txBytes.Serialize())
		if err != nil {
			t.Fatal(err)
		}

		code, err = s.RunFunc(txContext)
		if err != nil {
			t.Fatal(err)
		}

		if code.(uint64) != 0 {
			t.Fatalf("function exited with zero exit code: %d", code)
		}

		fromStoragePath := state.GetStorageHashForPath([]byte("balance"), make([]byte, 20))

		endAmount, _ := a.Get(fromStoragePath)

		if state.HashTo64(*endAmount) != 20 {
			t.Fatal("expected 10 PHR to be transferred to address 0")
		}

		endAmountFrom, _ := a.Get(storageHashPubkeyFrom)
		if state.HashTo64(*endAmountFrom) != 80 {
			t.Fatal("expected 90 PHR to be left in old address")
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransferShardRedeem(t *testing.T) {
	shardFile, err := os.Open("transfer_shard.wasm")
	if err != nil {
		t.Fatal(err)
	}

	shardCode, err := ioutil.ReadAll(shardFile)
	if err != nil {
		t.Fatal(err)
	}

	treeDB := csmt.NewInMemoryTreeDB()
	tree := csmt.NewTree(treeDB)
	s := state.NewFullShardState(tree)

	err = s.Update(func(a state.AccessInterface) error {
		pkBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2d4f87" +
			"20ee63e502ee2869afab7de234b80c")
		if err != nil {
			t.Fatal(err)
		}
		privkey, pubKey := secp256k1.PrivKeyFromBytes(pkBytes)

		var pubkeyFrom [33]byte
		copy(pubkeyFrom[:], pubKey.SerializeCompressed())

		var shardBytes [4]byte

		fromAddressHash := chainhash.HashH(append(shardBytes[:], pubkeyFrom[:]...))

		var fromAddress [20]byte
		copy(fromAddress[:], fromAddressHash[:20])

		s, err := state.NewShard(shardCode, []int64{8}, a, 0)
		if err != nil {
			t.Fatal(err)
		}

		redeemTx := RedeemTransaction{
			ToPubkeyHash: fromAddress,
		}

		redeemCtx, err := state.LoadArgumentContextFromTransaction(redeemTx.Serialize())
		if err != nil {
			t.Fatal(err)
		}

		code, err := s.RunFunc(redeemCtx)
		if err != nil {
			t.Fatal(err)
		}

		if code.(uint64) != 0 {
			t.Fatalf("function exited with non-zero exit code: %d", code)
		}

		tx := ShardTransaction{
			FromPubkeyHash: fromAddress,
			ToPubkeyHash:   [20]byte{},
			Amount:         10,
			Nonce:          0,
		}

		txBytes := tx.GetTransactionData()

		messageHash := chainhash.HashH(txBytes[:])

		sigBytes, err := secp256k1.SignCompact(privkey, messageHash[:], false)
		if err != nil {
			t.Fatal(err)
		}

		copy(tx.Signature[:], sigBytes)

		txContext, err := state.LoadArgumentContextFromTransaction(tx.Serialize())
		if err != nil {
			t.Fatal(err)
		}

		code, err = s.RunFunc(txContext)
		if err != nil {
			t.Fatal(err)
		}

		if code.(uint64) != 0 {
			t.Fatalf("function exited with non-zero exit code: %d", code)
		}

		toStoragePath := state.GetStorageHashForPath([]byte("balance"), make([]byte, 20))
		storageHashPubkeyFrom := state.GetStorageHashForPath([]byte("balance"), fromAddressHash[:20])

		endAmount, _ := a.Get(toStoragePath)

		if state.HashTo64(*endAmount) != 10 {
			t.Fatal("expected 10 PHR to be transferred to address 0")
		}

		endAmountFrom, _ := a.Get(storageHashPubkeyFrom)
		if state.HashTo64(*endAmountFrom) != 100000000-10 {
			t.Fatal("expected 90 PHR to be left in old address")
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkTransferShard(t *testing.B) {
	shardFile, err := os.Open("transfer_shard.wasm")
	if err != nil {
		t.Fatal(err)
	}

	shardCode, err := ioutil.ReadAll(shardFile)
	if err != nil {
		t.Fatal(err)
	}

	treeDB := csmt.NewInMemoryTreeDB()
	tree := csmt.NewTree(treeDB)
	store := state.NewFullShardState(tree)

	err = store.Update(func(a state.AccessInterface) error {
		pkBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2d4f87" +
			"20ee63e502ee2869afab7de234b80c")
		if err != nil {
			t.Fatal(err)
		}
		privkey, pubKey := secp256k1.PrivKeyFromBytes(pkBytes)

		var pubkeyFrom [33]byte
		copy(pubkeyFrom[:], pubKey.SerializeCompressed())

		var shardBytes [4]byte

		fromAddressHash := chainhash.HashH(append(shardBytes[:], pubkeyFrom[:]...))

		var fromAddress [20]byte
		copy(fromAddress[:], fromAddressHash[:20])

		tx := ShardTransaction{
			FromPubkeyHash: fromAddress,
			ToPubkeyHash:   [20]byte{},
			Amount:         10,
			Nonce:          0,
		}

		txBytes := tx.GetTransactionData()

		messageHash := chainhash.HashH(txBytes[:])

		sigBytes, err := secp256k1.SignCompact(privkey, messageHash[:], false)
		if err != nil {
			t.Fatal(err)
		}

		copy(tx.Signature[:], sigBytes)

		storageHashPubkeyFrom := state.GetStorageHashForPath([]byte("balance"), fromAddressHash[:20])

		// manually set the balance of from
		_ = a.Set(storageHashPubkeyFrom, state.Uint64ToHash(10*uint64(t.N)))

		txContext, err := state.LoadArgumentContextFromTransaction(tx.Serialize())
		if err != nil {
			t.Fatal(err)
		}

		s, err := state.NewShard(shardCode, []int64{8}, a, 0)
		if err != nil {
			t.Fatal(err)
		}

		t.ResetTimer()
		for i := 0; i < t.N; i++ {
			_, err := s.RunFunc(txContext)
			if err != nil {
				t.Fatal(err)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
