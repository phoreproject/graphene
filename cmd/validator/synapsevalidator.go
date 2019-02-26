package main

import (
	"encoding/binary"
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/phoreproject/synapse/beacon/config"

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
	p2pConnect := flag.String("p2pconnect", ":11783", "host and port for P2P rpc connection")
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
			validatorIndicesMap[i] = struct{}{}
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

	logrus.Info("connecting to blockchain RPC")

	blockchainConn, err := grpc.Dial(*beaconHost, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	logrus.Info("connecting to p2p RPC")
	p2pConn, err := grpc.Dial(*p2pConnect, grpc.WithInsecure())
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

	keys := make(map[uint32]*bls.SecretKey)

	for i := range vs {
		if _, found := validatorIndicesMap[i]; !found {
			continue
		}
		key := bls.DeserializeSecretKey(vs[i].PrivateKey)
		if err != nil {
			panic(err)
		}
		keys[uint32(i)] = &key
	}

	vm, err := validator.NewManager(blockchainConn, p2pConn, validatorIndices, validator.NewMemoryKeyStore(keys), &config.MainNetConfig)
	if err != nil {
		panic(err)
	}

	vm.Start()
}
