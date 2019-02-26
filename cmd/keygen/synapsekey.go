package main

import (
	"encoding/binary"
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/phoreproject/synapse/bls"

	"github.com/phoreproject/synapse/chainhash"
)

type hdReader struct {
	state chainhash.Hash
}

func (r *hdReader) Read(p []byte) (n int, err error) {
	length := len(p)
	sent := 0
	for sent < length {
		amountToSend := length - sent
		if amountToSend > chainhash.HashSize {
			amountToSend = chainhash.HashSize
		}
		copy(p[sent:sent+amountToSend], r.state[:])
		r.state = chainhash.HashH(r.state[:])
		sent += amountToSend
	}

	return sent, nil
}

func main() {
	rootkey := flag.String("rootkey", "", "this key derives all other keys")
	validators := flag.String("validators", "", "validator assignment")
	outfile := flag.String("outfile", "validators.pubs", "file with all of the public keys of validators")
	flag.Parse()

	validatorsStrings := strings.Split(*validators, ",")
	validatorIndices := []uint32{}
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

	rand := hdReader{chainhash.HashH([]byte(*rootkey))}

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
		key, _ := bls.RandSecretKey(&rand)

		pub := key.DerivePublicKey()

		var validatorIndexBytes [4]byte
		binary.BigEndian.PutUint32(validatorIndexBytes[:], v)
		_, err := f.Write(validatorIndexBytes[:])
		if err != nil {
			panic(err)
		}

		pubSer := pub.Serialize()
		_, err = f.Write(pubSer[:])
		if err != nil {
			panic(err)
		}
	}

	f.Close()
}
