package beacon_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/ssz"
)

func TestBlockIndex(t *testing.T) {
	bi := beacon.NewBlockIndex()

	genesisBlock := &primitives.Block{}
	genesisHash, _ := ssz.HashTreeRoot(genesisBlock)

	node, err := bi.AddBlockNodeToIndex(genesisBlock, genesisHash, chainhash.Hash{})
	if err != nil {
		t.Fatal(err)
	}

	g := chainhash.Hash(genesisHash)

	if !node.Hash.IsEqual(&g) {
		t.Fatal("expected hash to match")
	}

	node2, err := bi.AddBlockNodeToIndex(genesisBlock, genesisHash, chainhash.Hash{})
	if err != nil {
		t.Fatal(err)
	}

	if node != node2 {
		t.Fatal("expected add block node to be idempotent")
	}

	block1 := &primitives.Block{
		BlockHeader: primitives.BlockHeader{
			SlotNumber: 1,
			ParentRoot: chainhash.Hash{},
		},
	}
	block1Hash, _ := ssz.HashTreeRoot(block1)

	_, err = bi.AddBlockNodeToIndex(block1, block1Hash, chainhash.Hash{})
	if err != beacon.ErrNoParent {
		t.Fatal("expected block with no parent to error")
	}

	block1.BlockHeader.ParentRoot = genesisHash
	block1Hash, _ = ssz.HashTreeRoot(block1)

	node2, err = bi.AddBlockNodeToIndex(block1, block1Hash, chainhash.Hash{})
	if err != nil {
		t.Fatal(err)
	}

	b0disk := beacon.BlockNodeToDisk(*node)
	b1disk := beacon.BlockNodeToDisk(*node2)

	bi2 := beacon.NewBlockIndex()

	_, err = bi2.LoadBlockNode(&b0disk)
	if err != nil {
		t.Fatal(err)
	}
	_, err = bi2.LoadBlockNode(&b1disk)
	if err != nil {
		t.Fatal(err)
	}

	b1disk.Parent = chainhash.Hash{}
	_, err = bi2.LoadBlockNode(&b1disk)
	if err != beacon.ErrNoParent {
		t.Fatal("expected block with no parent to error")
	}
}

func TestChainGenesis(t *testing.T) {
	c := beacon.NewChain()

	genesisHash := chainhash.HashH([]byte("hello"))

	genesisNode := &beacon.BlockNode{
		Hash:      genesisHash,
		Height:    0,
		Slot:      0,
		RANDAO: chainhash.Hash{},

		Parent:    nil,
		StateRoot: chainhash.Hash{},
		Children:  nil,
	}

	if c.Genesis() != nil {
		t.Fatal("expected genesis block with no blocks defined to be nil")
	}

	if c.Tip() != nil {
		t.Fatal("expected tip with no blocks defined to be nil")
	}

	c.SetTip(genesisNode)

	if !c.Genesis().Hash.IsEqual(&genesisHash) {
		t.Fatal("expected genesis hash to match returned value")
	}

	if c.Height() != 0 {
		t.Fatal("expected genesis height to be 1")
	}

	if !c.Contains(genesisNode) {
		t.Fatal("empty chain should contain genesis node")
	}

	if c.GetBlockByHeight(0) != genesisNode {
		t.Fatal("expected block at height 0 to be genesis block")
	}

	if c.GetBlockByHeight(1) != nil {
		t.Fatal("expected block at height past tip to be nil")
	}

	if c.Tip() != genesisNode {
		t.Fatal("expected block at height past tip to be nil")
	}

	if c.Next(genesisNode) != nil {
		t.Fatal("expected next of genesis block to be nil")
	}

	c.SetTip(nil)
	if c.Tip() != nil {
		t.Fatal("set tip should set tip to nil")
	}

	node := c.GetBlockBySlot(1)
	if node != nil {
		t.Fatal("expected get block by slot to return nil for empty chain")
	}
}

func generateNodes(toHeight uint64) []*beacon.BlockNode {
	genesisNode := &beacon.BlockNode{
		Hash:      chainhash.HashH([]byte(fmt.Sprintf("block %d", 0))),
		Height:    0,
		Slot:      0,
		Parent:    nil,
		StateRoot: chainhash.Hash{},
		Children:  nil,
	}

	nodes := make([]*beacon.BlockNode, toHeight)
	nodes[0] = genesisNode

	for i := uint64(1); i < toHeight; i++ {
		nodes[i] = &beacon.BlockNode{
			Hash:      chainhash.HashH([]byte(fmt.Sprintf("block %d", i))),
			Height:    i,
			Slot:      2 * i,
			Parent:    nodes[i-1],
			StateRoot: chainhash.Hash{},
			Children:  nil,
		}

		nodes[i-1].Children = []*beacon.BlockNode{nodes[i]}
	}

	return nodes
}

func TestChainOneBlock(t *testing.T) {
	c := beacon.NewChain()

	genesisHash := chainhash.HashH([]byte("hello"))
	block1Hash := chainhash.HashH([]byte("hello1"))

	genesisNode := &beacon.BlockNode{
		Hash:      genesisHash,
		Height:    0,
		Slot:      0,
		Parent:    nil,
		StateRoot: chainhash.Hash{},
		Children:  nil,
	}

	block1 := &beacon.BlockNode{
		Hash:      block1Hash,
		Height:    1,
		Slot:      1,
		Parent:    genesisNode,
		StateRoot: chainhash.Hash{},
		Children:  nil,
	}

	genesisNode.Children = append(genesisNode.Children, block1)

	c.SetTip(block1)

	if c.Tip() != block1 {
		t.Fatal("expected chain tip to be block 1")
	}

	if c.Next(genesisNode) != block1 {
		t.Fatal("expected next of genesis to be block 1")
	}
}

const NumberBlocks = 200

func TestChainBlockchain(t *testing.T) {
	c := beacon.NewChain()

	nodes := generateNodes(NumberBlocks)

	c.SetTip(nodes[NumberBlocks-1])

	if c.Tip() != nodes[NumberBlocks-1] {
		t.Fatal("expected chain tip to be last block")
	}

	if nodes[NumberBlocks-1].GetAncestorAtHeight(140).Height != 140 {
		t.Fatal("expected to be able to get ancestor at height")
	}

	if nodes[NumberBlocks-1].GetAncestorAtSlot(140).Slot != 140 {
		t.Fatal("expected to be able to get ancestor at slot")
	}

	if nodes[NumberBlocks-1].GetAncestorAtSlot(900) != nil {
		t.Fatal("expected GetAncestorAtSlot to return nil if slot is over tip")
	}

	if c.GetBlockByHeight(140).Height != 140 {
		t.Fatal("expected chain to be able to get block by height")
	}

	node := c.GetBlockBySlot(140)
	if node == nil || node.Slot != 140 {
		t.Fatal("expected chain to be able to get block by slot")
	}

	node = c.GetBlockBySlot(140)
	if node.Slot != 140 {
		t.Fatal("expected chain to error if slot is out of bounds")
	}

	node = c.GetBlockBySlot(480)
	if node != c.Tip() {
		t.Fatal("out of bounds block slot should be tip")
	}

	locator := c.GetChainLocator()

	step := int64(1)
	current := int64(c.Tip().Height)
	for i, h := range locator {
		if current < 0 {
			current = 0
		}
		node := c.GetBlockByHeight(int(current))
		if !bytes.Equal(node.Hash[:], h) {
			t.Fatalf("expected locator hash %d to match hash of block %d: %s but got %s", i, node.Height, node.Hash, h)
		}

		current = int64(current) - step
		step *= 2
	}
}

func TestBlockchainView(t *testing.T) {
	view := beacon.NewBlockchainView()

	genesisBlock := &primitives.Block{}
	gBlockHash, _ := ssz.HashTreeRoot(genesisBlock)

	gNode, err := view.Index.AddBlockNodeToIndex(genesisBlock, gBlockHash, chainhash.Hash{})
	if err != nil {
		t.Fatal(err)
	}

	nodes := make([]*beacon.BlockNode, 200)
	nodes[0] = gNode

	for i := 1; i < 200; i++ {
		blockN := &primitives.Block{
			BlockHeader: primitives.BlockHeader{
				SlotNumber: uint64(i),
				ParentRoot: nodes[i-1].Hash,
			},
		}
		blockNHash, _ := ssz.HashTreeRoot(blockN)
		nNode, err := view.Index.AddBlockNodeToIndex(genesisBlock, blockNHash, chainhash.Hash{})
		if err != nil {
			t.Fatal(err)
		}

		nodes[i] = nNode
	}

	view.Chain.SetTip(nodes[199])

	if view.SetFinalizedHead(chainhash.Hash{}, primitives.State{}) {
		t.Fatal("expected setting finalized head to fail with wrong hash")
	}

	if view.SetJustifiedHead(chainhash.Hash{}, primitives.State{}) {
		t.Fatal("expected setting justified head to fail with wrong hash")
	}

	view.SetFinalizedHead(nodes[150].Hash, primitives.State{})
	view.SetJustifiedHead(nodes[140].Hash, primitives.State{})

	finalizedNode, _ := view.GetFinalizedHead()
	if finalizedNode != nodes[150] {
		t.Fatal("expected finalized node to be set properly")
	}

	justifiedNode, _ := view.GetJustifiedHead()
	if justifiedNode != nodes[140] {
		t.Fatal("expected justified node to be set properly")
	}
}
