package db_test

import (
	"crypto/rand"
	"fmt"
	"github.com/go-test/deep"
	"github.com/libp2p/go-libp2p-core/crypto"
	"testing"

	"github.com/phoreproject/synapse/ssz"

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

func TestLatestAttestations(t *testing.T) {
	imdb := db.NewInMemoryDB()

	att1 := primitives.Attestation{
		Data: primitives.AttestationData{
			Slot:                0,
			BeaconBlockHash:     chainhash.Hash{1},
			SourceEpoch:         2,
			SourceHash:          chainhash.Hash{3},
			TargetEpoch:         4,
			TargetHash:          chainhash.Hash{5},
			Shard:               6,
			LatestCrosslinkHash: chainhash.Hash{7},
			ShardBlockHash:      chainhash.Hash{8},
			ShardStateHash:      chainhash.Hash{9},
		},
		ParticipationBitfield: nil,
		CustodyBitfield:       nil,
		AggregateSig:          [48]byte{},
	}

	_ = imdb.SetLatestAttestationsIfNeeded([]uint32{1, 2, 3, 4, 5}, att1)

	att2, err := imdb.GetLatestAttestation(4)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(att1, *att2); diff != nil {
		t.Fatal(diff)
	}

	if _, err := imdb.GetLatestAttestation(6); err == nil {
		t.Fatal("expected attestation to error if not found")
	}

	att3 := primitives.Attestation{
		Data: primitives.AttestationData{
			Slot:                0,
			BeaconBlockHash:     chainhash.Hash{2},
			SourceEpoch:         2,
			SourceHash:          chainhash.Hash{3},
			TargetEpoch:         4,
			TargetHash:          chainhash.Hash{5},
			Shard:               6,
			LatestCrosslinkHash: chainhash.Hash{7},
			ShardBlockHash:      chainhash.Hash{8},
			ShardStateHash:      chainhash.Hash{9},
		},
		ParticipationBitfield: nil,
		CustodyBitfield:       nil,
		AggregateSig:          [48]byte{},
	}

	_ = imdb.SetLatestAttestationsIfNeeded([]uint32{1, 2, 3, 4, 5}, att3)

	if diff := deep.Equal(att1, *att2); diff != nil {
		t.Fatal(diff)
	}
}

func TestClose(t *testing.T) {
	imdb := db.NewInMemoryDB()

	err := imdb.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestHeadBlock(t *testing.T) {
	imdb := db.NewInMemoryDB()
	err := imdb.SetHeadBlock(chainhash.Hash{1})
	if err != nil {
		t.Fatal(err)
	}
	h, err := imdb.GetHeadBlock()
	if err != nil {
		t.Fatal(err)
	}

	if h[0] != 1 {
		t.Fatal("expected hash to match")
	}
}

func TestFinalizedHead(t *testing.T) {
	imdb := db.NewInMemoryDB()
	err := imdb.SetFinalizedHead(chainhash.Hash{1})
	if err != nil {
		t.Fatal(err)
	}
	h, err := imdb.GetFinalizedHead()
	if err != nil {
		t.Fatal(err)
	}

	if h[0] != 1 {
		t.Fatal("expected hash to match")
	}
}

func TestJustifiedHead(t *testing.T) {
	imdb := db.NewInMemoryDB()
	err := imdb.SetJustifiedHead(chainhash.Hash{1})
	if err != nil {
		t.Fatal(err)
	}
	h, err := imdb.GetJustifiedHead()
	if err != nil {
		t.Fatal(err)
	}

	if h[0] != 1 {
		t.Fatal("expected hash to match")
	}
}

func TestHostKey(t *testing.T) {
	imdb := db.NewInMemoryDB()

	priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	err = imdb.SetHostKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	priv2, err := imdb.GetHostKey()
	if err != nil {
		t.Fatal(err)
	}

	if !priv.Equals(priv2) {
		t.Fatal("expected private keys to match")
	}
}

func TestTransactionalUpdate(t *testing.T) {
	imdb := db.NewInMemoryDB()
	o := 0

	err := imdb.TransactionalUpdate(func(transaction interface{}) error {
		o = 1
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if o != 1 {
		t.Fatal("expected callback to be run")
	}
}

func TestBlockNodes(t *testing.T) {
	imdb := db.NewInMemoryDB()

	n1 := db.BlockNodeDisk{
		Hash:      chainhash.Hash{1},
		Height:    2,
		Slot:      3,
		Parent:    chainhash.Hash{4},
		StateRoot: chainhash.Hash{5},
		Children:  nil,
	}
	err := imdb.SetBlockNode(n1)
	if err != nil {
		t.Fatal(err)
	}

	n2, err := imdb.GetBlockNode(n1.Hash)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := imdb.GetBlockNode(chainhash.Hash{}); err == nil {
		t.Fatal("did not error with nonexistent block node")
	}

	if diff := deep.Equal(n1, *n2); diff != nil {
		t.Fatal(diff)
	}
}

func TestGenesisTime(t *testing.T) {
	imdb := db.NewInMemoryDB()

	err := imdb.SetGenesisTime(100)
	if err != nil {
		t.Fatal(err)
	}
	g2, err := imdb.GetGenesisTime()
	if err != nil {
		t.Fatal(err)
	}

	if g2 != 100 {
		t.Fatal("expected genesis time to match")
	}
}

func TestJustifiedState(t *testing.T) {
	imdb := db.NewInMemoryDB()

	state := primitives.State{
		Slot: 1,
	}

	err := imdb.SetJustifiedState(state)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := imdb.GetJustifiedState()
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(*s2, state); diff != nil {
		t.Fatal(diff)
	}
}

func TestFinalizedState(t *testing.T) {
	imdb := db.NewInMemoryDB()

	state := primitives.State{
		Slot: 1,
	}

	err := imdb.SetFinalizedState(state)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := imdb.GetFinalizedState()
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(*s2, state); diff != nil {
		t.Fatal(diff)
	}
}
