package blockchain_test

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"

	"github.com/phoreproject/synapse/blockchain"
	"github.com/phoreproject/synapse/primitives"
)

var zeroHash = chainhash.Hash{}

func TestReorganization(t *testing.T) {
	b := blockchain.NewBlockchain(blockchain.NewBlockIndex())

	h00 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{zeroHash}}
	h00hash := h00.Hash()
	h01 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h00hash}}
	h01hash := h01.Hash()
	h02 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h01hash}}
	h02hash := h02.Hash()
	h03 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h02hash}}
	h03hash := h03.Hash()
	h04 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h03hash}}
	h04hash := h04.Hash()
	h05 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h04hash}}

	b.AddBlock(h00)
	b.AddBlock(h01)
	b.AddBlock(h02)
	b.AddBlock(h03)
	b.AddBlock(h04)
	b.AddBlock(h05)

	if b.Height() != 5 {
		t.Errorf("Height is not expected value of 6; got %d", b.Height())
	}

	h13 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h02hash}}
	h13hash := h13.Hash()
	h14 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h13hash}}
	h14hash := h14.Hash()
	h15 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h14hash}}
	h15hash := h15.Hash()
	h16 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h15hash}}

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

	h00 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{zeroHash}}
	h00hash := h00.Hash()
	h01 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h00hash}}
	h01hash := h01.Hash()
	h02 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h01hash}}
	h02hash := h02.Hash()
	h03 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h02hash}}
	h03hash := h03.Hash()
	h04 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h03hash}}
	h04hash := h04.Hash()
	h05 := primitives.BlockHeader{AncestorHashes: []chainhash.Hash{h04hash}}

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
