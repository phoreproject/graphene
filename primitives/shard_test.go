package primitives_test

import (
	"github.com/go-test/deep"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"testing"
)

func TestShardBlockHeader_Copy(t *testing.T) {
	baseHeader := &primitives.ShardBlockHeader{
		PreviousBlockHash:   chainhash.Hash{},
		Slot:                0,
		Signature:           [48]byte{},
		StateRoot:           chainhash.Hash{},
		TransactionRoot:     chainhash.Hash{},
		FinalizedBeaconHash: chainhash.Hash{},
	}

	copyHeader := baseHeader.Copy()
	copyHeader.PreviousBlockHash[0] = 1

	if baseHeader.PreviousBlockHash[0] == 1 {
		t.Fatal("mutating copy previousBlockHash mutated base")
	}

	copyHeader.Signature[0] = 1

	if baseHeader.Signature[0] == 1 {
		t.Fatal("mutating copy signature mutated base")
	}

	copyHeader.StateRoot[0] = 1

	if baseHeader.StateRoot[0] == 1 {
		t.Fatal("mutating copy stateRoot mutated base")
	}

	copyHeader.TransactionRoot[0] = 1

	if baseHeader.TransactionRoot[0] == 1 {
		t.Fatal("mutating copy transactionRoot mutated base")
	}

	copyHeader.FinalizedBeaconHash[0] = 1

	if baseHeader.FinalizedBeaconHash[0] == 1 {
		t.Fatal("mutating copy finalizedBeaconHash mutated base")
	}

	copyHeader.Slot = 1

	if baseHeader.Slot == 1 {
		t.Fatal("mutating copy slot mutated base")
	}
}

func TestShardBlockHeaderToFromProto(t *testing.T) {
	baseHeader := &primitives.ShardBlockHeader{
		PreviousBlockHash:   chainhash.Hash{1},
		Slot:                1,
		Signature:           [48]byte{1},
		StateRoot:           chainhash.Hash{1},
		TransactionRoot:     chainhash.Hash{1},
		FinalizedBeaconHash: chainhash.Hash{1},
	}

	baseHeaderProto := baseHeader.ToProto()
	fromProto, err := primitives.ShardBlockHeaderFromProto(baseHeaderProto)
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(fromProto, baseHeader); diff != nil {
		t.Fatal(diff)
	}
}

func TestShardTransaction_Copy(t *testing.T) {
	baseTransaction := &primitives.ShardTransaction{
		TransactionData: []byte{0},
	}

	copyTransaction := baseTransaction.Copy()
	copyTransaction.TransactionData[0] = 1

	if baseTransaction.TransactionData[0] == 1 {
		t.Fatal("mutating copy transaction data mutated base")
	}
}

func TestTransactionToFromProto(t *testing.T) {
	baseTransaction := &primitives.ShardTransaction{
		TransactionData: []byte{1},
	}

	baseTransactionProto := baseTransaction.ToProto()
	fromProto, err := primitives.ShardTransactionFromProto(baseTransactionProto)
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(fromProto, baseTransaction); diff != nil {
		t.Fatal(diff)
	}
}

func TestShardBlockBody_Copy(t *testing.T) {
	baseTransaction := &primitives.ShardBlockBody{
		Transactions: []primitives.ShardTransaction{
			{
				TransactionData: []byte{0},
			},
		},
	}

	copyTransaction := baseTransaction.Copy()
	copyTransaction.Transactions[0].TransactionData = nil

	if baseTransaction.Transactions[0].TransactionData == nil {
		t.Fatal("mutating copy transaction data mutated base")
	}
}

func TestShardBlockBodyToFromProto(t *testing.T) {
	baseBody := &primitives.ShardBlockBody{
		Transactions: []primitives.ShardTransaction{
			{
				TransactionData: []byte{1},
			},
		},
	}

	baseBodyProto := baseBody.ToProto()
	fromProto, err := primitives.ShardBlockBodyFromProto(baseBodyProto)
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(fromProto, baseBody); diff != nil {
		t.Fatal(diff)
	}
}

func TestShardBlock_Copy(t *testing.T) {
	baseBlock := &primitives.ShardBlock{
		Header: primitives.ShardBlockHeader{
			PreviousBlockHash:   chainhash.Hash{},
			Slot:                0,
			Signature:           [48]byte{},
			StateRoot:           chainhash.Hash{},
			TransactionRoot:     chainhash.Hash{},
			FinalizedBeaconHash: chainhash.Hash{},
		},
		Body: primitives.ShardBlockBody{
			Transactions: []primitives.ShardTransaction{
				{},
			},
		},
	}

	copyBlock := baseBlock.Copy()
	copyBlock.Header.Slot = 1

	if baseBlock.Header.Slot == 1 {
		t.Fatal("mutating copy blockHeader mutated base")
	}

	copyBlock.Body.Transactions = nil

	if baseBlock.Body.Transactions == nil {
		t.Fatal("mutating copy blockBody mutated base")
	}
}

func TestShardBlockToFromProto(t *testing.T) {
	baseBlock := &primitives.ShardBlock{
		Header: primitives.ShardBlockHeader{
			PreviousBlockHash:   chainhash.Hash{1},
			Slot:                1,
			Signature:           [48]byte{1},
			StateRoot:           chainhash.Hash{1},
			TransactionRoot:     chainhash.Hash{1},
			FinalizedBeaconHash: chainhash.Hash{1},
		},
		Body: primitives.ShardBlockBody{
			Transactions: []primitives.ShardTransaction{
				{
					TransactionData: []byte{1},
				},
			},
		},
	}

	baseBlockProto := baseBlock.ToProto()
	fromProto, err := primitives.ShardBlockFromProto(baseBlockProto)
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(fromProto, baseBlock); diff != nil {
		t.Fatal(diff)
	}
}
