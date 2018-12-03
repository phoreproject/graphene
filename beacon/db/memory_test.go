package db_test

import (
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/transaction"

	"github.com/phoreproject/synapse/beacon/primitives"
	"github.com/phoreproject/synapse/beacon/db"
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
		Attestations: []transaction.AttestationRecord{
			{
				Data: transaction.AttestationSignedData{
					Slot:  0,
					Shard: 1,
					ParentHashes: []chainhash.Hash{
						{},
						{},
						{},
						{},
					},
					ShardBlockHash:             chainhash.Hash{},
					LastCrosslinkHash:          chainhash.Hash{},
					ShardBlockCombinedDataRoot: chainhash.Hash{},
					JustifiedSlot:              0,
				},
				AttesterBitfield: []uint8{},
				PoCBitfield:      []uint8{},
				AggregateSig:     bls.Signature{},
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
