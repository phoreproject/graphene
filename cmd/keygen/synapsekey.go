package main

import (
	"encoding/binary"
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/chainhash"

	"github.com/phoreproject/synapse/beacon"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/validator"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

func main() {
	rootkey := flag.String("rootkey", "", "this key derives all other keys")
	validators := flag.String("validators", "", "validator assignment")
	outfile := flag.String("outfile", "validators.pubs", "file with all of the public keys of validators")
	flag.Parse()

	validatorsStrings := strings.Split(*validators, ",")
	var validatorIndices []uint32
	for _, s := range validatorsStrings {
		if !strings.ContainsRune(s, '-') {
			i, err := strconv.Atoi(s)
			if err != nil {
				panic("invalid validators parameter")
			}
			validatorIndices = append(validatorIndices, uint32(i))
		} else {
			parts := strings.SplitN(s, "-", 2)
			if len(parts) != 2 {
				panic("invalid validators parameter")
			}
			first, err := strconv.Atoi(parts[0])
			if err != nil {
				panic("invalid validators parameter")
			}
			second, err := strconv.Atoi(parts[1])
			if err != nil {
				panic("invalid validators parameter")
			}
			for i := first; i <= second; i++ {
				validatorIndices = append(validatorIndices, uint32(i))
			}
		}
	}

	f, err := os.Create(*outfile)
	if err != nil {
		panic(err)
	}
	var numValidatorBytes [4]byte
	binary.BigEndian.PutUint32(numValidatorBytes[:], uint32(len(validatorIndices)))
	_, err = f.Write(numValidatorBytes[:])
	if err != nil {
		panic(err)
	}

	for _, v := range validatorIndices {
		key, _ := bls.RandSecretKey(validator.GetReaderForID(*rootkey, v))

		pub := key.DerivePublicKey()

		var validatorIndexBytes [4]byte
		binary.BigEndian.PutUint32(validatorIndexBytes[:], v)
		_, err := f.Write(validatorIndexBytes[:])
		if err != nil {
			panic(err)
		}

		pubSer := pub.Serialize()

		h, err := ssz.TreeHash(pubSer)
		if err != nil {
			panic(err)
		}

		sig, err := bls.Sign(key, h[:], bls.DomainDeposit)
		if err != nil {
			panic(err)
		}

		sigSer := sig.Serialize()

		iv := beacon.InitialValidatorEntry{
			PubKey:                pubSer,
			ProofOfPossession:     sigSer,
			WithdrawalShard:       0,
			WithdrawalCredentials: chainhash.Hash{},
			DepositSize:           config.MainNetConfig.MaxDeposit,
		}

		err = binary.Write(f, binary.BigEndian, iv)
		if err != nil {
			panic(err)
		}
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}
}
