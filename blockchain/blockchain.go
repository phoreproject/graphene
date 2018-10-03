package blockchain

import (
	"encoding/binary"
	"errors"
	"io"
	"time"

	"github.com/phoreproject/synapse/serialization"
	"github.com/phoreproject/synapse/transaction"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// BlockHeader is a block header for the beacon chain.
type BlockHeader struct {
	ParentHash      *chainhash.Hash
	SlotNumber      uint64
	RandaoReveal    *chainhash.Hash
	ActiveStateRoot *chainhash.Hash
	Timestamp       time.Time
	TransactionRoot *chainhash.Hash
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

// GetBlockNodeByHash gets a block node by the given hash from the index.
func (b BlockIndex) GetBlockNodeByHash(h chainhash.Hash) (*BlockNode, error) {
	o, err := b.index[h]
	if err {
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
	index           BlockIndex
	bestChainHeight uint64
	chain           []*BlockNode
}

// AddBlock adds a block header to the current chain.
func (b Blockchain) AddBlock(h BlockHeader) error {
	parent, err := b.index.GetBlockNodeByHash(*h.ParentHash)
	if err != nil {
		return err
	}

	node := &BlockNode{BlockHeader: h, PrevNode: parent, Height: parent.Height + 1}

	// Add block to the index
	b.index.AddNode(node)

	// TODO: flawed fork choice mechanism (should be IMD-GHOST)
	b.chain = append(b.chain, node)
	return nil
}
