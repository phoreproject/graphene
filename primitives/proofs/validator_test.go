package proofs_test

import (
	"testing"
	"time"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/csmt"
	"github.com/phoreproject/synapse/primitives"
	"github.com/phoreproject/synapse/primitives/proofs"
	"github.com/phoreproject/synapse/ssz"
	"github.com/phoreproject/synapse/validator"
	"github.com/sirupsen/logrus"
)

// SetupState initializes state with a certain number of initial validators
func SetupState(initialValidators int, c *config.Config) (*primitives.State, validator.Keystore, error) {
	keystore := validator.NewFakeKeyStore()

	var validators []primitives.InitialValidatorEntry

	for i := 0; i < initialValidators; i++ {
		key := keystore.GetKeyForValidator(uint32(i))
		pub := key.DerivePublicKey()
		hashPub, err := ssz.HashTreeRoot(pub.Serialize())
		if err != nil {
			return nil, nil, err
		}
		proofOfPossession, err := bls.Sign(key, hashPub[:], bls.DomainDeposit)
		if err != nil {
			return nil, nil, err
		}
		validators = append(validators, primitives.InitialValidatorEntry{
			PubKey:                pub.Serialize(),
			ProofOfPossession:     proofOfPossession.Serialize(),
			WithdrawalShard:       1,
			WithdrawalCredentials: chainhash.Hash{},
			DepositSize:           c.MaxDeposit,
		})
	}

	s, err := primitives.InitializeState(c, validators, uint64(time.Now().Unix()), false)
	return s, &keystore, err
}

func TestValidatorProof(t *testing.T) {
	c := &config.RegtestConfig

	logrus.SetLevel(logrus.ErrorLevel)

	state, _, err := SetupState(c.ShardCount*c.TargetCommitteeSize*2+5, c)
	if err != nil {
		t.Fatal(err)
	}

	rootHash := proofs.GetValidatorHash(state)

	vw, err := proofs.ConstructValidatorProof(state, 10)
	if err != nil {
		t.Fatal(err)
	}

	if !csmt.CheckWitness(&vw.Proof, rootHash) {
		t.Fatal("proof did not verify")
	}
}
