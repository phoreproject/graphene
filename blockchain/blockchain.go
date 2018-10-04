package blockchain

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/phoreproject/synapse/serialization"
	"github.com/phoreproject/synapse/transaction"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

var zeroHash = chainhash.Hash{}

// BlockHeader is a block header for the beacon chain.
type BlockHeader struct {
	ParentHash      chainhash.Hash
	SlotNumber      uint64
	RandaoReveal    chainhash.Hash
	ActiveStateRoot chainhash.Hash
	Timestamp       time.Time
	TransactionRoot chainhash.Hash
}

// Serialize serializes a block header to bytes.
func (b BlockHeader) Serialize() []byte {
	var slotNumBytes [8]byte
	var timeBytes [8]byte
	binary.BigEndian.PutUint64(timeBytes[:], uint64(b.Timestamp.Unix()))
	binary.BigEndian.PutUint64(slotNumBytes[:], b.SlotNumber)
	return serialization.AppendAll(b.ParentHash[:], slotNumBytes[:], b.RandaoReveal[:], b.ActiveStateRoot[:], timeBytes[:], b.TransactionRoot[:])
}

// Deserialize reads a block header from the given reader.
func (b BlockHeader) Deserialize(r io.Reader) error {
	p, err := serialization.ReadHash(r)
	if err != nil {
		return err
	}
	b.ParentHash = p
	slotNumber, err := serialization.ReadUint64(r)
	if err != nil {
		return err
	}
	b.SlotNumber = slotNumber
	rr, err := serialization.ReadHash(r)
	if err != nil {
		return err
	}
	b.RandaoReveal = rr
	asr, err := serialization.ReadHash(r)
	if err != nil {
		return err
	}
	b.ActiveStateRoot = asr
	t, err := serialization.ReadUint64(r)
	if err != nil {
		return err
	}
	b.Timestamp = time.Unix(int64(t), 0)
	tr, err := serialization.ReadHash(r)
	if err != nil {
		return err
	}
	b.TransactionRoot = tr
	return nil
}

// Hash gets the hash of a block node.
func (b BlockHeader) Hash() chainhash.Hash {
	return transaction.GetHash(b)
}

// BlockNode is a block header with a reference to the
// last block.
type BlockNode struct {
	BlockHeader
	Height   uint64
	PrevNode *BlockNode
}

// BlockIndex is an in-memory store of block headers.
type BlockIndex struct {
	index map[chainhash.Hash]*BlockNode
}

// NewBlockIndex creates and initializes a new block index.
func NewBlockIndex() BlockIndex {
	return BlockIndex{index: make(map[chainhash.Hash]*BlockNode)}
}

// GetBlockNodeByHash gets a block node by the given hash from the index.
func (b BlockIndex) GetBlockNodeByHash(h chainhash.Hash) (*BlockNode, error) {
	o, found := b.index[h]
	if !found {
		return nil, errors.New("could not find block in index")
	}
	return o, nil
}

// AddNode adds a node to the block index.
func (b BlockIndex) AddNode(node *BlockNode) {
	h := transaction.GetHash(node.BlockHeader)
	b.index[h] = node
}

// Blockchain represents a chain of blocks.
type Blockchain struct {
	index BlockIndex
	chain []*BlockNode
}

func NewBlockchain(index BlockIndex) Blockchain {
	return Blockchain{index: index}
}

// AddBlock adds a block header to the current chain. The block should already
// have been validated by this point.
func (b *Blockchain) AddBlock(h BlockHeader) error {
	var parent *BlockNode
	if !(h.ParentHash == zeroHash && len(b.chain) == 0) {
		p, err := b.index.GetBlockNodeByHash(h.ParentHash)
		parent = p
		if err != nil {
			return err
		}
	}

	height := uint64(0)
	if parent != nil {
		height = parent.Height + 1
	}

	node := &BlockNode{BlockHeader: h, PrevNode: parent, Height: height}

	// Add block to the index
	b.index.AddNode(node)

	b.UpdateChainHead(node)

	return nil
}

// UpdateChainHead updates the blockchain head if needed
func (b *Blockchain) UpdateChainHead(n *BlockNode) {
	if int64(n.Height) > int64(len(b.chain)-1) {
		b.SetTip(n)
	}
}

// SetTip sets the tip of the chain.
func (b *Blockchain) SetTip(n *BlockNode) {
	fmt.Println("test!")
	needed := n.Height + 1
	if uint64(cap(b.chain)) < needed {
		nodes := make([]*BlockNode, needed, needed+100)
		copy(nodes, b.chain)
		b.chain = nodes
	} else {
		prevLen := int32(len(b.chain))
		b.chain = b.chain[0:needed]
		for i := prevLen; uint64(i) < needed; i++ {
			b.chain[i] = nil
		}
	}

	for n != nil && b.chain[n.Height] != n {
		b.chain[n.Height] = n
		n = n.PrevNode
	}
}

// Tip returns the block at the tip of the chain.
func (b Blockchain) Tip() *BlockNode {
	return b.chain[len(b.chain)-1]
}

// GetNodeByHeight gets a node from the active blockchain by height.
func (b Blockchain) GetNodeByHeight(height int64) *BlockNode {
	return b.chain[height]
}

// Height returns the height of the chain.
func (b Blockchain) Height() int {
	return len(b.chain) - 1
}
