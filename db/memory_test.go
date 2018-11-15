package db_test

import (
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/transaction"

	"github.com/phoreproject/synapse/db"
	"github.com/phoreproject/synapse/primitives"
)

func TestStoreRetrieve(t *testing.T) {
	db := db.NewInMemoryDB()

	ancestorHashes := make([]chainhash.Hash, 32)

	for i := range ancestorHashes {
		ancestorHashes[i] = chainhash.HashH([]byte(fmt.Sprintf("test %d", i)))
	}

	b := primitives.Block{
		SlotNumber:            0,
		RandaoReveal:          chainhash.HashH([]byte("test")),
		AncestorHashes:        ancestorHashes,
		ActiveStateRoot:       chainhash.Hash{},
		CrystallizedStateRoot: chainhash.Hash{},
		Specials:              []transaction.Transaction{},
		Attestations: []transaction.Attestation{
			{
				Slot:    0,
				ShardID: 1,
				ObliqueParentHashes: []chainhash.Hash{
					chainhash.Hash{},
					chainhash.Hash{},
					chainhash.Hash{},
					chainhash.Hash{},
				},
			},
		},
	}

	err := db.SetBlock(b)
	if err != nil {
		t.Fatal(err)
	}

	b1, err := db.GetBlockForHash(b.Hash())
	if err != nil {
		t.Fatalf("could not find block hash %x", b.Hash())
	}

	if b.Hash() != b1.Hash() {
		t.Fatalf("block hashes do not match (expected: %x, returned: %x)", b.Hash(), b1.Hash())
	}

	_, err = db.GetBlockForHash(chainhash.Hash{})
	if err == nil {
		t.Fatalf("incorrectly found blockhash")
	}
}
