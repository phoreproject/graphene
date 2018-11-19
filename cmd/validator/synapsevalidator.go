package main

import (
	"flag"
	"strconv"
	"strings"

	"github.com/phoreproject/synapse/validator"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

func main() {
	logrus.Info("Starting validator manager")
	beaconHost := flag.String("beaconhost", ":11782", "the address to connect to the beacon node")
	validators := flag.String("validators", "", "validators to manage (id separated by commas) (ex. \"1,2,3\")")
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

	logrus.WithField("validators", *validators).Debug("running with validators")

	conn, err := grpc.Dial(*beaconHost, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	vm, err := validator.NewManager(conn, validatorIndices, &validator.FakeKeyStore{})
	if err != nil {
		panic(err)
	}

	vm.Start()
}
