package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/chainhash"

	"github.com/phoreproject/synapse/bls"
)

type xorshift struct {
	state uint64
}

func newXORShift(state uint64) *xorshift {
	return &xorshift{state}
}

func (xor *xorshift) Read(b []byte) (int, error) {
	for i := range b {
		x := xor.state
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		b[i] = uint8(x)
		xor.state = x
	}
	return len(b), nil
}

func main() {
	numKeys := flag.Uint("numkeys", 5000, "how many keys to generate")
	outFile := flag.String("out", "keypairs.bin", "where to put out file")
	flag.Parse()

	fmt.Printf("Starting generating %d keys\n", numKeys)

	x := newXORShift(2)
	keys := make([]bls.SecretKey, *numKeys)
	validators := make([]beacon.InitialValidatorEntry, *numKeys)
	for i := range keys {
		k, err := bls.RandSecretKey(x)
		if err != nil {
			panic(err)
		}
		keys[i] = *k
	}

	var wg sync.WaitGroup

	wg.Add(len(keys))

	for i := range keys {

		go func(keyNum int) {
			key := keys[keyNum]
			pub := key.DerivePublicKey()
			hashPub, err := ssz.TreeHash(pub.Serialize())
			if err != nil {
				panic(err)
			}
			proofOfPossession, err := bls.Sign(&key, hashPub[:], bls.DomainDeposit)
			if err != nil {
				panic(err)
			}
			validators[keyNum] = beacon.InitialValidatorEntry{
				PubKey:                pub.Serialize(),
				ProofOfPossession:     proofOfPossession.Serialize(),
				WithdrawalShard:       0,
				WithdrawalCredentials: chainhash.Hash{},
				DepositSize:           config.MainNetConfig.MaxDeposit * config.UnitInCoin,
			}
			fmt.Printf("Generating key %d\n", keyNum)
			wg.Done()
		}(i)
	}

	wg.Wait()

	var lengthBytes [4]byte
	binary.BigEndian.PutUint32(lengthBytes[:], uint32(*numKeys))

	f, err := os.Create(*outFile)

	if err != nil {
		panic(err)
	}

	_, err = f.Write(lengthBytes[:])
	if err != nil {
		panic(err)
	}

	for _, v := range validators {
		err := binary.Write(f, binary.BigEndian, v)
		if err != nil {
			panic(err)
		}
	}

	f.Close()
}
