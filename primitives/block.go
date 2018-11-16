package primitives

import (
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	pb "github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/transaction"
)

// To create a block:
// - increase slot number
// - reveal randao commitment
// - update ancestor hashes
// - calculate state roots
// - aggregate specials + attestations (mempool?)

// Block represents a single beacon chain block.
type Block struct {
	SlotNumber            uint64
	RandaoReveal          chainhash.Hash
	AncestorHashes        []chainhash.Hash
	ActiveStateRoot       chainhash.Hash
	CrystallizedStateRoot chainhash.Hash
	Specials              []transaction.Transaction
	Attestations          []transaction.Attestation
}

// Hash gets the hash of the block header
func (b *Block) Hash() chainhash.Hash {
	m, _ := proto.Marshal(b.ToProto())
	return chainhash.HashH(m)
}

// BlockFromProto creates a block from the protobuf block given
func BlockFromProto(blockProto *pb.Block) (*Block, error) {
	if len(blockProto.AncestorHashes) != 32 {
		return nil, fmt.Errorf("ancestor hashes length incorrect. got: %d, expected: 32", len(blockProto.AncestorHashes))
	}

	ancestorHashes := make([]chainhash.Hash, 32)
	for i := range blockProto.AncestorHashes {
		a, err := chainhash.NewHash(blockProto.AncestorHashes[i])
		if err != nil {
			return nil, err
		}
		ancestorHashes[i] = *a
	}

	activeStateRoot, err := chainhash.NewHash(blockProto.ActiveStateRoot)
	if err != nil {
		return nil, err
	}

	randaoReveal, err := chainhash.NewHash(blockProto.RandaoReveal)
	if err != nil {
		return nil, err
	}

	crystallizedStateRoot, err := chainhash.NewHash(blockProto.CrystallizedStateRoot)
	if err != nil {
		return nil, err
	}

	specials := make([]transaction.Transaction, len(blockProto.Specials))
	for i := range blockProto.Specials {
		s := blockProto.Specials[i]
		tx, err := transaction.DeserializeTransaction(s.Type, s.Data)
		if err != nil {
			return nil, err
		}
		specials[i] = *tx
	}

	attestations := make([]transaction.Attestation, len(blockProto.Attestations))
	for i := range blockProto.Attestations {
		a := blockProto.Attestations[i]
		att, err := transaction.NewAttestationFromProto(a)
		if err != nil {
			return nil, err
		}
		attestations[i] = *att
	}

	return &Block{
		SlotNumber:            blockProto.SlotNumber,
		RandaoReveal:          *randaoReveal,
		AncestorHashes:        ancestorHashes,
		ActiveStateRoot:       *activeStateRoot,
		CrystallizedStateRoot: *crystallizedStateRoot,
		Specials:              specials,
		Attestations:          attestations,
	}, nil
}

// ToProto gets the protobuf representation of block
func (b *Block) ToProto() *pb.Block {
	ancestorHashes := make([][]byte, len(b.AncestorHashes))
	for i := range ancestorHashes {
		ancestorHashes[i] = b.AncestorHashes[i][:]
	}

	specials := make([]*pb.Special, len(b.Specials))
	for i := range specials {
		s := b.Specials[i].Serialize()
		specials[i] = s
	}

	attestations := make([]*pb.Attestation, len(b.Attestations))
	for i := range attestations {
		attestations[i] = b.Attestations[i].ToProto()
	}

	return &pb.Block{
		SlotNumber:            b.SlotNumber,
		RandaoReveal:          b.RandaoReveal[:],
		AncestorHashes:        ancestorHashes,
		ActiveStateRoot:       b.ActiveStateRoot[:],
		CrystallizedStateRoot: b.CrystallizedStateRoot[:],
		Specials:              specials,
		Attestations:          attestations,
	}
}
