package transfer

import (
	"encoding/binary"
	"github.com/phoreproject/synapse/shard/execution"
)

// ShardTransaction is a transaction for the transfer shard.
type ShardTransaction struct {
	FromPubkey   [33]byte
	Signature    [65]byte
	ToPubkeyHash [32]byte
	Amount       uint64
}

// Serialize serializes the transfer transaction to bytes.
func (t *ShardTransaction) Serialize() []byte {
	var amountBytes [8]byte
	binary.BigEndian.PutUint64(amountBytes[:], t.Amount)

	out, _ := execution.SerializeTransactionWithArguments("transfer_to_address", t.FromPubkey[:], t.Signature[:], t.ToPubkeyHash[:], amountBytes[:])

	return out
}

// Code is the binary code used.
var Code = FSMustByte(false, "/transfer_shard.wasm")
