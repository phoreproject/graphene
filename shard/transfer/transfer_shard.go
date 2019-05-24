package transfer

import "encoding/binary"

// ShardContext is the context for a call to Transfer.
type ShardContext struct {
	FromPubkey   [33]byte
	Signature    [65]byte
	ToPubkeyHash [32]byte
	Amount       uint64
}

// LoadArgument loads an argument from the context.
func (t ShardContext) LoadArgument(argumentNumber int32) []byte {
	switch argumentNumber {
	case 0:
		return t.FromPubkey[:]
	case 1:
		return t.Signature[:]
	case 2:
		return t.ToPubkeyHash[:]
	case 3:
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, t.Amount)
		return b
	}

	return nil
}
