package state

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"

	"github.com/phoreproject/synapse/csmt"

	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/phoreproject/synapse/chainhash"
)

func TestShard(t *testing.T) {
	shardFile, err := os.Open("test_shard.wasm")
	if err != nil {
		t.Fatal(err)
	}

	shardCode, err := ioutil.ReadAll(shardFile)
	if err != nil {
		t.Fatal(err)
	}

	treeDB := csmt.NewInMemoryTreeDB()
	tree := csmt.NewTree(treeDB)

	store := NewFullShardState(tree)

	err = store.Update(func(a AccessInterface) error {
		s, err := NewShard(shardCode, []int64{2}, a, 0)
		if err != nil {
			t.Fatal(err)
		}

		err = s.RunFunc(NewEmptyContext("run"))
		if err != nil {
			t.Fatal(err)
		}

		err = s.RunFunc(NewEmptyContext("run"))
		if err != nil {
			t.Fatal(err)
		}

		addr0, err := s.Storage.Get(Uint64ToHash(0))
		if err != nil {
			t.Fatal(err)
		}

		if HashTo64(*addr0) != 2 {
			t.Fatalf("Expected to load 2 from Phore storage, got: %d", addr0)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkShardCall(b *testing.B) {
	shardFile, err := os.Open("test_shard.wasm")
	if err != nil {
		b.Fatal(err)
	}

	shardCode, err := ioutil.ReadAll(shardFile)
	if err != nil {
		b.Fatal(err)
	}

	treeDB := csmt.NewInMemoryTreeDB()
	tree := csmt.NewTree(treeDB)

	store := NewFullShardState(tree)

	err = store.Update(func(a AccessInterface) error {
		s, err := NewShard(shardCode, []int64{2}, a, 0)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err = s.RunFunc(NewEmptyContext("run"))
			if err != nil {
				b.Fatal(err)
			}
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
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

	treeDB := csmt.NewInMemoryTreeDB()
	tree := csmt.NewTree(treeDB)
	err = tree.Update(func(a csmt.TreeTransactionAccess) error {
		s, err := NewShard(shardCode, []int64{2}, a, 0)
		if err != nil {
			t.Fatal(err)
		}

		err = s.RunFunc(NewEmptyContext("run"))
		if err != nil {
			t.Fatal(err)
		}

		addr0, err := s.Storage.Get(Uint64ToHash(0))
		if err != nil {
			t.Fatal(err)
		}

		if HashTo64(*addr0) != 0 {
			t.Fatalf("Expected to load 0 from Phore storage, got: %d", addr0)
		}

		addr1, err := s.Storage.Get(Uint64ToHash(1))
		if err != nil {
			t.Fatal(err)
		}

		if byte(HashTo64(*addr1)) != chainhash.HashH([]byte{1, 2, 3, 4})[0] {
			t.Fatal("Expected to load correct hash value from shard")
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSignatureVerification(t *testing.T) {
	// Decode a hex-encoded private key.
	pkBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2d4f87" +
		"20ee63e502ee2869afab7de234b80c")
	if err != nil {
		t.Fatal(err)
	}
	privKey, pubKey := secp256k1.PrivKeyFromBytes(pkBytes)

	// Sign a message using the private key.
	message := "test message"
	messageHash := chainhash.HashH([]byte(message))
	signature, err := secp256k1.SignCompact(privKey, messageHash[:], false)
	if err != nil {
		t.Fatal(err)
	}

	var signatureFixed [65]byte
	copy(signatureFixed[:], signature)

	sig := DecompressSignature(signatureFixed)

	// Verify the signature for the message using the public key.
	verified := sig.Verify(messageHash[:], pubKey)
	if !verified {
		t.Fatal("ECDSA signature did not verify")
	}

	pub, _, err := secp256k1.RecoverCompact(signature, messageHash[:])
	if err != nil {
		t.Fatal(err)
	}

	if !pub.IsEqual(pubKey) {
		t.Fatal("expected recovered pubkey to match original pubkey")
	}
}

func BenchmarkShardECDSA(b *testing.B) {
	shardFile, err := os.Open("test_ecdsa_shard.wasm")
	if err != nil {
		b.Fatal(err)
	}

	shardCode, err := ioutil.ReadAll(shardFile)
	if err != nil {
		b.Fatal(err)
	}

	treeDB := csmt.NewInMemoryTreeDB()
	tree := csmt.NewTree(treeDB)

	err = tree.Update(func(tx csmt.TreeTransactionAccess) error {
		s, err := NewShard(shardCode, []int64{2}, tx, 0)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err = s.RunFunc(NewEmptyContext("run"))
			if err != nil {
				b.Fatal(err)
			}
		}

		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
}
