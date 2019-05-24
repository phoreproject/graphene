package transfer

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/shard/execution"
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

	store := execution.NewMemoryStorage()

	pkBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2d4f87" +
		"20ee63e502ee2869afab7de234b80c")
	if err != nil {
		t.Fatal(err)
	}
	privkey, pubKey := secp256k1.PrivKeyFromBytes(pkBytes)

	var pubkeyFrom [33]byte
	copy(pubkeyFrom[:], pubKey.SerializeCompressed())

	hashPubkeyFrom := chainhash.HashH(pubkeyFrom[:])

	zeroHash := chainhash.Hash{}

	message := fmt.Sprintf("transfer %d PHR to %s", 10, zeroHash.String())

	messageHash := chainhash.HashH([]byte(message))

	var signature [65]byte

	sigBytes, err := secp256k1.SignCompact(privkey, messageHash[:], false)
	if err != nil {
		t.Fatal(err)
	}

	copy(signature[:], sigBytes)

	// manually set the balance of from
	store.PhoreStore64(hashPubkeyFrom, 100)

	ctx := ShardContext{
		FromPubkey:   pubkeyFrom,
		Signature:    signature,
		ToPubkeyHash: zeroHash,
		Amount:       10,
	}

	s, err := execution.NewShard(shardCode, []int64{8}, store, ctx)
	if err != nil {
		t.Fatal(err)
	}

	code, err := s.RunFunc("transfer_to_address")
	if err != nil {
		t.Fatal(err)
	}

	if code.(uint64) != 0 {
		t.Fatal("function exited with non-zero exit code")
	}

	endAmount := store.PhoreLoad64(zeroHash)
	if endAmount != 10 {
		t.Fatal("expected 10 PHR to be transferred to address 0")
	}

	endAmountFrom := store.PhoreLoad64(hashPubkeyFrom)
	if endAmountFrom != 90 {
		t.Fatal("expected 90 PHR to be left in old address")
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

	store := execution.NewMemoryStorage()

	pkBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2d4f87" +
		"20ee63e502ee2869afab7de234b80c")
	if err != nil {
		t.Fatal(err)
	}
	privkey, pubKey := secp256k1.PrivKeyFromBytes(pkBytes)

	var pubkeyFrom [33]byte
	copy(pubkeyFrom[:], pubKey.SerializeCompressed())

	hashPubkeyFrom := chainhash.HashH(pubkeyFrom[:])

	zeroHash := chainhash.Hash{}

	message := fmt.Sprintf("transfer %d PHR to %s", 10, zeroHash.String())

	messageHash := chainhash.HashH([]byte(message))

	var signature [65]byte

	sigBytes, err := secp256k1.SignCompact(privkey, messageHash[:], false)
	if err != nil {
		t.Fatal(err)
	}

	copy(signature[:], sigBytes)

	// manually set the balance of from
	store.PhoreStore64(hashPubkeyFrom, 10*uint64(t.N))

	ctx := ShardContext{
		FromPubkey:   pubkeyFrom,
		Signature:    signature,
		ToPubkeyHash: zeroHash,
		Amount:       10,
	}

	s, err := execution.NewShard(shardCode, []int64{8}, store, ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		_, err := s.RunFunc("transfer_to_address")
		if err != nil {
			t.Fatal(err)
		}
	}
}
