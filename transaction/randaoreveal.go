package transaction

import "github.com/btcsuite/btcd/chaincfg/chainhash"

// RandaoRevealTransaction is used to update a RANDAO
type RandaoRevealTransaction struct {
	ValidatorIndex uint32
	Commitment     chainhash.Hash
}
