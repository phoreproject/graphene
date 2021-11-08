package transfer

import (
	"encoding/binary"

	"github.com/phoreproject/graphene/shard/state"
)

// ShardTransaction is a transaction for the transfer shard.
type ShardTransaction struct {
	FromPubkeyHash [20]byte
	Signature      [65]byte
	ToPubkeyHash   [20]byte
	Amount         uint64
	Nonce          uint32
}

// GetTransactionData gets the transaction data.
func (t *ShardTransaction) GetTransactionData() [33]byte {
	var amountBytes [8]byte
	binary.BigEndian.PutUint64(amountBytes[:], t.Amount)

	var txData [33]byte
	txData[0] = 0
	binary.BigEndian.PutUint32(txData[1:5], t.Nonce)

	copy(txData[5:25], t.ToPubkeyHash[:])
	binary.BigEndian.PutUint64(txData[25:33], t.Amount)

	return txData
}

// Serialize serializes the transfer transaction to bytes.
func (t *ShardTransaction) Serialize() []byte {

	data := t.GetTransactionData()

	out, _ := state.SerializeTransactionWithArguments("transfer_to_address", data[:], t.Signature[:], t.FromPubkeyHash[:])

	return out
}

// RedeemTransaction redeems a premine from the transfer shard.
type RedeemTransaction struct {
	ToPubkeyHash [20]byte
}

// Serialize serializes the transfer transaction to bytes.
func (t *RedeemTransaction) Serialize() []byte {
	out, _ := state.SerializeTransactionWithArguments("redeem_premine", t.ToPubkeyHash[:])

	return out
}

// Code is the binary code used.
var Code = FSMustByte(false, "/transfer_shard.wasm")
