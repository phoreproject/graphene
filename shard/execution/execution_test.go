package execution

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/phoreproject/synapse/chainhash"
)

func TestShard(t *testing.T) {
	shardFile, err := os.Open("transfer_shard.wasm")
	if err != nil {
		t.Fatal(err)
	}

	shardCode, err := ioutil.ReadAll(shardFile)
	if err != nil {
		t.Fatal(err)
	}

	store := NewMemoryStorage()

	s, err := NewShard(shardCode, []int64{2}, store)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.RunFunc(3)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.RunFunc(3)
	if err != nil {
		t.Fatal(err)
	}

	addr0 := s.Storage.PhoreLoad(0)

	if addr0 != 2 {
		t.Fatalf("Expected to load 2 from Phore storage, got: %d", addr0)
	}
}

func BenchmarkShardCall(b *testing.B) {
	shardFile, err := os.Open("transfer_shard.wasm")
	if err != nil {
		b.Fatal(err)
	}

	shardCode, err := ioutil.ReadAll(shardFile)
	if err != nil {
		b.Fatal(err)
	}

	store := NewMemoryStorage()

	s, err := NewShard(shardCode, []int64{2}, store)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err = s.RunFunc(3)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestECDSAShard(t *testing.T) {
	shardFile, err := os.Open("test_ecdsa_shard.wasm")
	if err != nil {
		t.Fatal(err)
	}

	shardCode, err := ioutil.ReadAll(shardFile)
	if err != nil {
		t.Fatal(err)
	}

	store := NewMemoryStorage()

	s, err := NewShard(shardCode, []int64{2}, store)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.RunFunc(3)
	if err != nil {
		t.Fatal(err)
	}

	addr0 := s.Storage.PhoreLoad(0)

	if addr0 != 1 {
		t.Fatalf("Expected to load 1 from Phore storage, got: %d", addr0)
	}
}

func TestSignatureVerification(t *testing.T) {
	// Decode a hex-encoded private key.
	pkBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2d4f87" +
		"20ee63e502ee2869afab7de234b80c")
	if err != nil {
		fmt.Println(err)
		return
	}
	privKey, pubKey := secp256k1.PrivKeyFromBytes(pkBytes)

	// Sign a message using the private key.
	message := "test message"
	messageHash := chainhash.HashH([]byte(message))
	signature, err := secp256k1.SignCompact(privKey, messageHash[:], false)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Serialize and display the signature.
	fmt.Printf("Serialized Signature: %x\n", signature)

	var signatureFixed [65]byte
	copy(signatureFixed[:], signature)

	sig := DecompressSignature(signatureFixed)

	// Verify the signature for the message using the public key.
	verified := sig.Verify(messageHash[:], pubKey)
	fmt.Printf("Signature Verified? %v\n", verified)
	fmt.Println(messageHash)

	pub, _, err := secp256k1.RecoverCompact(signature, messageHash[:])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Pubkey: %x\n", pub.Serialize())

	if !pub.IsEqual(pubKey) {
		t.Fatal("expected recovered pubkey to match original pubkey")
	}
}
