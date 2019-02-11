package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sync"

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
	keysSer := make([][64]byte, *numKeys)
	pubs := make([][96]byte, *numKeys)
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
			pubs[keyNum] = key.DerivePublicKey().Serialize()
			copy(keysSer[keyNum][:], key.Serialize())
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

	for i := range keys {
		_, err := f.Write(keysSer[i][:])
		if err != nil {
			panic(err)
		}
		_, err = f.Write(pubs[i][:])
		if err != nil {
			panic(err)
		}
	}

	f.Close()
}
