package primitives

import (
	"errors"
	"fmt"
	"github.com/prysmaticlabs/go-ssz"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
)

// ShardBlockHeader is the header for blocks on shard chains.
type ShardBlockHeader struct {
	PreviousBlockHash   chainhash.Hash
	Slot                uint64
	Signature           [48]byte
	StateRoot           chainhash.Hash
	TransactionRoot     chainhash.Hash
	FinalizedBeaconHash chainhash.Hash
}

// ToProto gets the protobuf representation of the shard block header.
func (sb *ShardBlockHeader) ToProto() *pb.ShardBlockHeader {
	return &pb.ShardBlockHeader{
		PreviousBlockHash:   sb.PreviousBlockHash[:],
		Slot:                sb.Slot,
		Signature:           sb.Signature[:],
		StateRoot:           sb.StateRoot[:],
		TransactionRoot:     sb.TransactionRoot[:],
		FinalizedBeaconHash: sb.FinalizedBeaconHash[:],
	}
}

// ShardBlockHeaderFromProto converts a protobuf representation of a shard block header to a shard block header.
func ShardBlockHeaderFromProto(header *pb.ShardBlockHeader) (*ShardBlockHeader, error) {
	if len(header.Signature) > 48 {
		return nil, errors.New("signature should be 48 bytes long")
	}
	newHeader := &ShardBlockHeader{
		Slot: header.Slot,
	}

	copy(newHeader.Signature[:], header.Signature)

	err := newHeader.PreviousBlockHash.SetBytes(header.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	err = newHeader.StateRoot.SetBytes(header.StateRoot)
	if err != nil {
		return nil, err
	}

	err = newHeader.TransactionRoot.SetBytes(header.TransactionRoot)
	if err != nil {
		return nil, err
	}

	err = newHeader.FinalizedBeaconHash.SetBytes(header.FinalizedBeaconHash)
	if err != nil {
		return nil, err
	}

	return newHeader, nil
}

// Copy returns a copy of the shard block header.
func (sb *ShardBlockHeader) Copy() ShardBlockHeader {
	return *sb
}

// ShardTransaction is a single transaction on a shard chain.
type ShardTransaction struct {
	TransactionData []byte
}

// ToProto converts a shard transaction to protobuf format.
func (st *ShardTransaction) ToProto() *pb.ShardTransaction {
	return &pb.ShardTransaction{
		TransactionData: st.TransactionData,
	}
}

// ShardTransactionFromProto converts a protobuf representation of a shard transaction to a shard transaction.
func ShardTransactionFromProto(st *pb.ShardTransaction) (*ShardTransaction, error) {
	return &ShardTransaction{
		TransactionData: st.TransactionData,
	}, nil
}

// Copy returns a copy of the shard trnasaction.
func (st *ShardTransaction) Copy() ShardTransaction {
	tx := ShardTransaction{
		TransactionData: make([]byte, len(st.TransactionData)),
	}

	copy(tx.TransactionData, st.TransactionData)

	return tx
}

// ShardBlockBody is the body of a single block on a shard chain.
type ShardBlockBody struct {
	Transactions []ShardTransaction
}

// ToProto converts a shard block body to protobuf format.
func (sb *ShardBlockBody) ToProto() *pb.ShardBlockBody {
	txs := make([]*pb.ShardTransaction, len(sb.Transactions))

	for i := range txs {
		txs[i] = sb.Transactions[i].ToProto()
	}
	return &pb.ShardBlockBody{
		Transactions: txs,
	}
}

// ShardBlockBodyFromProto converts a protobuf representation of shard block body to a shard block body.
func ShardBlockBodyFromProto(sb *pb.ShardBlockBody) (*ShardBlockBody, error) {
	txs := make([]ShardTransaction, len(sb.Transactions))

	for i := range txs {
		tx, err := ShardTransactionFromProto(sb.Transactions[i])
		if err != nil {
			return nil, err
		}

		txs[i] = *tx
	}

	return &ShardBlockBody{
		Transactions: txs,
	}, nil
}

// Copy returns a copy of the shard block header.
func (sb *ShardBlockBody) Copy() ShardBlockBody {
	newTransactions := make([]ShardTransaction, len(sb.Transactions))

	for i := range newTransactions {
		newTransactions[i] = sb.Transactions[i].Copy()
	}

	return ShardBlockBody{
		Transactions: newTransactions,
	}
}

// ShardBlock is a single block on a shard chain.
type ShardBlock struct {
	Header ShardBlockHeader
	Body   ShardBlockBody
}

// ToProto converts a shard block body to protobuf format.
func (sb *ShardBlock) ToProto() *pb.ShardBlock {
	return &pb.ShardBlock{
		Header: sb.Header.ToProto(),
		Body:   sb.Body.ToProto(),
	}
}

// ShardBlockFromProto converts a protobuf representation of shard block to a shard block.
func ShardBlockFromProto(sb *pb.ShardBlock) (*ShardBlock, error) {
	header, err := ShardBlockHeaderFromProto(sb.Header)
	if err != nil {
		return nil, err
	}

	body, err := ShardBlockBodyFromProto(sb.Body)
	if err != nil {
		return nil, err
	}

	return &ShardBlock{
		Header: *header,
		Body:   *body,
	}, nil
}

// Copy returns a deep copy of the shard block.
func (sb *ShardBlock) Copy() ShardBlock {
	return ShardBlock{
		Header: sb.Header.Copy(),
		Body:   sb.Body.Copy(),
	}
}

// GetGenesisBlockForShard gets the genesis block for a certain shard.
func GetGenesisBlockForShard(shardID uint64) ShardBlock {
	var transactions []ShardTransaction

	transactionRoot, err := ssz.HashTreeRoot(transactions)
	if err != nil {
		panic(err)
	}

	return ShardBlock{
		Header: ShardBlockHeader{
			PreviousBlockHash:   chainhash.HashH([]byte(fmt.Sprintf("%d", shardID))),
			Slot:                0,
			Signature:           [48]byte{},
			StateRoot:           EmptyTree,
			TransactionRoot:     transactionRoot,
			FinalizedBeaconHash: chainhash.Hash{},
		},
		Body: ShardBlockBody{
			Transactions: nil,
		},
	}
}
