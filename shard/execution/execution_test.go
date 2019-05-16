package execution

import (
	"io/ioutil"
	"os"
	"testing"
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
