package blockchain_test

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"

	"github.com/phoreproject/synapse/blockchain"
)

func TestReorganization(t *testing.T) {
	b := blockchain.NewBlockchain(blockchain.NewBlockIndex())

	h00 := blockchain.BlockHeader{}
	h00hash := h00.Hash()
	h01 := blockchain.BlockHeader{ParentHash: h00hash}
	h01hash := h01.Hash()
	h02 := blockchain.BlockHeader{ParentHash: h01hash}
	h02hash := h02.Hash()
	h03 := blockchain.BlockHeader{ParentHash: h02hash}
	h03hash := h03.Hash()
	h04 := blockchain.BlockHeader{ParentHash: h03hash}
	h04hash := h04.Hash()
	h05 := blockchain.BlockHeader{ParentHash: h04hash}

	b.AddBlock(h00)
	b.AddBlock(h01)
	b.AddBlock(h02)
	b.AddBlock(h03)
	b.AddBlock(h04)
	b.AddBlock(h05)

	if b.Height() != 5 {
		t.Errorf("Height is not expected value of 6; got %d", b.Height())
	}

	h13 := blockchain.BlockHeader{ParentHash: h02hash}
	h13hash := h13.Hash()
	h14 := blockchain.BlockHeader{ParentHash: h13hash}
	h14hash := h14.Hash()
	h15 := blockchain.BlockHeader{ParentHash: h14hash}
	h15hash := h15.Hash()
	h16 := blockchain.BlockHeader{ParentHash: h15hash}

	b.AddBlock(h13)

	if b.Tip().Hash() != h05.Hash() {
		t.Errorf("Tip is set incorrectly after adding a forked block that shouldn't reorg.")
	}

	b.AddBlock(h14)
	b.AddBlock(h15)

	if b.Tip().Hash() != h05.Hash() {
		t.Errorf("Tip is set incorrectly after adding a forked block that shouldn't reorg.")
	}

	b.AddBlock(h16)

	if b.Tip().Hash() != h16.Hash() {
		t.Errorf("Tip is set incorrectly after adding a forked block that shouldn't reorg.")
	}
}

func TestHeightConsistency(t *testing.T) {
	b := blockchain.NewBlockchain(blockchain.NewBlockIndex())

	h00 := blockchain.BlockHeader{}
	h00hash := h00.Hash()
	h01 := blockchain.BlockHeader{ParentHash: h00hash}
	h01hash := h01.Hash()
	h02 := blockchain.BlockHeader{ParentHash: h01hash}
	h02hash := h02.Hash()
	h03 := blockchain.BlockHeader{ParentHash: h02hash}
	h03hash := h03.Hash()
	h04 := blockchain.BlockHeader{ParentHash: h03hash}
	h04hash := h04.Hash()
	h05 := blockchain.BlockHeader{ParentHash: h04hash}

	b.AddBlock(h00)
	b.AddBlock(h01)
	b.AddBlock(h02)
	b.AddBlock(h03)
	b.AddBlock(h04)
	b.AddBlock(h05)

	block5 := b.GetNodeByHeight(5)
	if block5.Height != 5 {
		t.Errorf("block height doesn't match getter")
	}
}

func TestBlockHeaderSerializeAndDeserialize(t *testing.T) {
	b := blockchain.BlockHeader{
		ParentHash:      chainhash.HashH([]byte("random")),
		RandaoReveal:    chainhash.HashH([]byte("random1")),
		ActiveStateRoot: chainhash.HashH([]byte("random2")),
		TransactionRoot: chainhash.HashH([]byte("random3")),
		Timestamp:       time.Unix(time.Now().Unix(), 0),
		SlotNumber:      rand.Uint64(),
	}

	bBytes := b.Serialize()

	newB := blockchain.BlockHeader{}
	err := newB.Deserialize(bytes.NewBuffer(bBytes))
	if err != nil {
		t.Error(err)
	}
	if newB != b {
		t.Error("invalid serialization")
	}
}
