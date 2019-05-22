package main

import (
	"flag"
	"strconv"
	"strings"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/utils"

	"github.com/phoreproject/synapse/validator"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	logrus.Info("Starting validator manager")
	beaconHost := flag.String("beaconhost", ":11782", "the address to connect to the beacon node")
	validators := flag.String("validators", "", "validators to manage (id separated by commas) (ex. \"1,2,3\")")
	rootkey := flag.String("rootkey", "testnet", "root key to run validators")
	flag.Parse()

	utils.CheckNTP()

	validatorsStrings := strings.Split(*validators, ",")
	var validatorIndices []uint32
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

	vm, err := validator.NewManager(blockchainConn, validatorIndices, validator.NewRootKeyStore(*rootkey), &config.MainNetConfig)
	if err != nil {
		panic(err)
	}

	vm.Start()
}
