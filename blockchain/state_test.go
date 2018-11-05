package blockchain_test

import (
	"crypto/rand"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/phoreproject/synapse/bls"

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

func GenerateNextBlock(b *blockchain.Blockchain) primitives.Block {
	return primitives.Block{}
}

func TestStateActiveValidatorChanges(t *testing.T) {
	var randaoSecret [32]byte

	rand.Read(randaoSecret[:])

	randaoCommitment := chainhash.HashH(randaoSecret[:])

	validators := []blockchain.InitialValidatorEntry{}

	for i := 0; i <= 100000; i++ {
		validators = append(validators, blockchain.InitialValidatorEntry{
			PubKey:            bls.PublicKey{},
			ProofOfPossession: bls.Signature{},
			WithdrawalShard:   1,
			WithdrawalAddress: serialization.Address{},
			RandaoCommitment:  randaoCommitment,
		})
	}

	b, err := blockchain.NewBlockchainWithInitialValidators(db.NewInMemoryDB(), &blockchain.MainNetConfig, validators)
	if err != nil {
		t.Error(err)
		return
	}

	lastBlock, err := b.GetNodeByHeight(0)
	if err != nil {
		t.Error(err)
		return
	}

	block1 := primitives.Block{
		SlotNumber:            1,
		RandaoReveal:          zeroHash,
		AncestorHashes:        generateAncestorHashes([]chainhash.Hash{lastBlock}),
		ActiveStateRoot:       zeroHash,
		CrystallizedStateRoot: zeroHash,
		Specials:              []transaction.Transaction{},
		Attestations:          []transaction.Attestation{},
	}

	err = b.ProcessBlock(&block1)
	if err != nil {
		t.Error(err)
		return
	}

	s := b.GetState()

	spew.Println(s.Crystallized.ShardAndCommitteeForSlots)
	if len(s.Crystallized.ShardAndCommitteeForSlots[0]) == 0 {
		t.Errorf("invalid initial validator entries")
	}
}
