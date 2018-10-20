package blockchain_test

import (
	"crypto/rand"
	"testing"

	"github.com/phoreproject/synapse/serialization"
	"github.com/phoreproject/synapse/transaction"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/phoreproject/synapse/blockchain"
	"github.com/phoreproject/synapse/db"
	"github.com/phoreproject/synapse/primitives"
)

func generateAncestorHashes(hashes []chainhash.Hash) []chainhash.Hash {
	var out [32]chainhash.Hash

	for i := range out {
		if i <= len(hashes)-1 {
			out[i] = hashes[i]
		} else {
			out[i] = zeroHash
		}
	}

	return out[:]
}

func TestStateActiveValidatorChanges(t *testing.T) {
	b := blockchain.NewBlockchain(db.NewInMemoryDB(), &blockchain.MainNetConfig)

	var randaoSecret [32]byte

	rand.Read(randaoSecret[:])

	randaoCommitment := chainhash.HashH(randaoSecret[:])

	validators := []blockchain.InitialValidatorEntry{}

	for i := 0; i <= 5*128; i++ {
		validators = append(validators, blockchain.InitialValidatorEntry{
			PubKey:            []byte{},
			ProofOfPossession: []byte{},
			WithdrawalShard:   1,
			WithdrawalAddress: serialization.Address{},
			RandaoCommitment:  randaoCommitment,
		})
	}

	b.InitializeState(validators)

	block0 := primitives.Block{
		SlotNumber:            0,
		RandaoReveal:          zeroHash,
		AncestorHashes:        generateAncestorHashes([]chainhash.Hash{}),
		ActiveStateRoot:       zeroHash,
		CrystallizedStateRoot: zeroHash,
		Specials:              []transaction.Transaction{},
		Attestations:          []transaction.Attestation{},
	}

	b.ProcessBlock(&block0)
}
