package db_test

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/go-ssz"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"

	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/primitives"
)

func TestStoreRetrieve(t *testing.T) {
	imdb := db.NewInMemoryDB()

	ancestorHashes := make([]chainhash.Hash, 32)

	for i := range ancestorHashes {
		ancestorHashes[i] = chainhash.HashH([]byte(fmt.Sprintf("test %d", i)))
	}

	b := primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber:   0,
			ParentRoot:   chainhash.Hash{},
			StateRoot:    chainhash.Hash{},
			RandaoReveal: bls.EmptySignature.Serialize(),
			Signature:    bls.EmptySignature.Serialize(),
		},
		BlockBody: primitives.BlockBody{
			Attestations:      []primitives.Attestation{},
			ProposerSlashings: []primitives.ProposerSlashing{},
			CasperSlashings:   []primitives.CasperSlashing{},
			Deposits:          []primitives.Deposit{},
			Exits:             []primitives.Exit{},
		},
	}

	err := imdb.SetBlock(b)
	if err != nil {
		t.Fatal(err)
	}

	blockHash, err := ssz.HashTreeRoot(b)
	if err != nil {
		t.Fatal(err)
	}
	b1, err := imdb.GetBlockForHash(blockHash)
	if err != nil {
		t.Fatalf("could not find block hash %x", blockHash)
	}

	blockHashAfter, err := ssz.HashTreeRoot(b1)
	if err != nil {
		t.Fatal(err)
	}

	if blockHash != blockHashAfter {
		t.Fatalf("block hashes do not match (expected: %x, returned: %x)", blockHash, blockHashAfter)
	}

	_, err = imdb.GetBlockForHash(chainhash.Hash{})
	if err == nil {
		t.Fatalf("incorrectly found blockhash")
	}
}
