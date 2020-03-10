package db_test

import (
	"crypto/rand"
	"fmt"
	"github.com/go-test/deep"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/go-ssz"
	"io/ioutil"
	"os"
	"testing"
)

func TestBadgerStoreRetrieve(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)

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

	err = bdb.SetBlock(b)
	if err != nil {
		t.Fatal(err)
	}

	blockHash, err := ssz.HashTreeRoot(b)
	if err != nil {
		t.Fatal(err)
	}
	b1, err := bdb.GetBlockForHash(blockHash)
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

	_, err = bdb.GetBlockForHash(chainhash.Hash{})
	if err == nil {
		t.Fatalf("incorrectly found blockhash")
	}

	if err := bdb.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBadgerLatestAttestations(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)

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

	_ = bdb.SetLatestAttestationsIfNeeded([]uint32{1, 2, 3, 4, 5}, att1)

	att2, err := bdb.GetLatestAttestation(4)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(att1, *att2); diff != nil {
		t.Fatal(diff)
	}

	if _, err := bdb.GetLatestAttestation(6); err == nil {
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

	_ = bdb.SetLatestAttestationsIfNeeded([]uint32{1, 2, 3, 4, 5}, att3)

	if diff := deep.Equal(att1, *att2); diff != nil {
		t.Fatal(diff)
	}

	if err := bdb.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBadgerHeadBlock(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)
	err = bdb.SetHeadBlock(chainhash.Hash{1})
	if err != nil {
		t.Fatal(err)
	}
	h, err := bdb.GetHeadBlock()
	if err != nil {
		t.Fatal(err)
	}

	if h[0] != 1 {
		t.Fatal("expected hash to match")
	}

	if err := bdb.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBadgerFinalizedHead(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)
	err = bdb.SetFinalizedHead(chainhash.Hash{1})
	if err != nil {
		t.Fatal(err)
	}
	h, err := bdb.GetFinalizedHead()
	if err != nil {
		t.Fatal(err)
	}

	if h[0] != 1 {
		t.Fatal("expected hash to match")
	}

	if err := bdb.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBadgerJustifiedHead(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)
	err = bdb.SetJustifiedHead(chainhash.Hash{1})
	if err != nil {
		t.Fatal(err)
	}
	h, err := bdb.GetJustifiedHead()
	if err != nil {
		t.Fatal(err)
	}

	if h[0] != 1 {
		t.Fatal("expected hash to match")
	}

	if err := bdb.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBadgerHostKey(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)

	priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	err = bdb.SetHostKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	priv2, err := bdb.GetHostKey()
	if err != nil {
		t.Fatal(err)
	}

	if !priv.Equals(priv2) {
		t.Fatal("expected private keys to match")
	}

	if err := bdb.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBadgerTransactionalUpdate(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)
	o := 0

	err = bdb.TransactionalUpdate(func(transaction interface{}) error {
		o = 1
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if o != 1 {
		t.Fatal("expected callback to be run")
	}

	if err := bdb.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBadgerBlockNodes(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)

	n1 := db.BlockNodeDisk{
		Hash:      chainhash.Hash{1},
		Height:    2,
		Slot:      3,
		Parent:    chainhash.Hash{4},
		StateRoot: chainhash.Hash{5},
		Children:  []chainhash.Hash{},
	}
	err = bdb.SetBlockNode(n1)
	if err != nil {
		t.Fatal(err)
	}

	n2, err := bdb.GetBlockNode(n1.Hash)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := bdb.GetBlockNode(chainhash.Hash{}); err == nil {
		t.Fatal("did not error with nonexistent block node")
	}

	if diff := deep.Equal(n1, *n2); diff != nil {
		t.Fatal(diff)
	}

	if err := bdb.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBadgerGenesisTime(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)

	err = bdb.SetGenesisTime(100)
	if err != nil {
		t.Fatal(err)
	}
	g2, err := bdb.GetGenesisTime()
	if err != nil {
		t.Fatal(err)
	}

	if g2 != 100 {
		t.Fatal("expected genesis time to match")
	}
}

func TestBadgerJustifiedState(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)

	state := primitives.State{
		Slot:                      1,
		ValidatorRegistry:         []primitives.Validator{},
		ValidatorBalances:         []uint64{},
		ShardAndCommitteeForSlots: [][]primitives.ShardAndCommittee{},
		LatestCrosslinks:          []primitives.Crosslink{},
		PreviousCrosslinks:        []primitives.Crosslink{},
		LatestBlockHashes:         []chainhash.Hash{},
		CurrentEpochAttestations:  []primitives.PendingAttestation{},
		PreviousEpochAttestations: []primitives.PendingAttestation{},
		BatchedBlockRoots:         []chainhash.Hash{},
		ShardRegistry:             []chainhash.Hash{},
		Proposals:                 []primitives.ActiveProposal{},
		PendingVotes:              []primitives.AggregatedVote{},
	}

	err = bdb.SetJustifiedState(state)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := bdb.GetJustifiedState()
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(*s2, state); diff != nil {
		t.Fatal(diff)
	}

	if err := bdb.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBadgerFinalizedState(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger")
	if err != nil {
		t.Fatal(err)
	}
	bdb := db.NewBadgerDB(dir)

	state := primitives.State{
		Slot:                      1,
		ValidatorRegistry:         []primitives.Validator{},
		ValidatorBalances:         []uint64{},
		ShardAndCommitteeForSlots: [][]primitives.ShardAndCommittee{},
		LatestCrosslinks:          []primitives.Crosslink{},
		PreviousCrosslinks:        []primitives.Crosslink{},
		LatestBlockHashes:         []chainhash.Hash{},
		CurrentEpochAttestations:  []primitives.PendingAttestation{},
		PreviousEpochAttestations: []primitives.PendingAttestation{},
		BatchedBlockRoots:         []chainhash.Hash{},
		ShardRegistry:             []chainhash.Hash{},
		Proposals:                 []primitives.ActiveProposal{},
		PendingVotes:              []primitives.AggregatedVote{},
	}

	err = bdb.SetFinalizedState(state)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := bdb.GetFinalizedState()
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(*s2, state); diff != nil {
		t.Fatal(diff)
	}

	if err := bdb.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
}
