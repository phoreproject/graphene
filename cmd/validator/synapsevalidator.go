package main

import (
	"encoding/binary"
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/phoreproject/synapse/bls"

	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/validator"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	logrus.Info("Starting validator manager")
	beaconHost := flag.String("beaconhost", ":11782", "the address to connect to the beacon node")
	validators := flag.String("validators", "", "validators to manage (id separated by commas) (ex. \"1,2,3\")")
	fakeValidatorKeyStore := flag.String("fakevalidatorkeys", "keypairs.bin", "key file of fake validators")
	flag.Parse()

	validatorsStrings := strings.Split(*validators, ",")
	validatorIndices := []uint32{}
	validatorIndicesMap := map[int]struct{}{}
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
				validatorIndicesMap[i] = struct{}{}
			}
		}
	}

	logrus.WithField("validators", *validators).Debug("running with validators")

	conn, err := grpc.Dial(*beaconHost, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	f, err := os.Open(*fakeValidatorKeyStore)
	if err != nil {
		panic(err)
	}

	var lengthBytes [4]byte

	_, err = f.Read(lengthBytes[:])
	if err != nil {
		panic(err)
	}

	length := binary.BigEndian.Uint32(lengthBytes[:])

	vs := make([]beacon.InitialValidatorEntryAndPrivateKey, length)

	for i := uint32(0); i < length; i++ {
		var iv beacon.InitialValidatorEntryAndPrivateKey
		binary.Read(f, binary.BigEndian, &iv)
		vs[i] = iv
	}

	keys := make([]*bls.SecretKey, length)

	for i := range vs {
		if _, found := validatorIndicesMap[i]; !found {
			continue
		}
		key := bls.DeserializeSecretKey(vs[i].PrivateKey[:])
		if err != nil {
			panic(err)
		}
		keys[i] = &key
	}

	vm, err := validator.NewManager(conn, validatorIndices, validator.NewMemoryKeyStore(keys))
	if err != nil {
		panic(err)
	}

	vm.Start()
}
