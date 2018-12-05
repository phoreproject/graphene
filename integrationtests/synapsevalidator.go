package integrationtests

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/phoreproject/synapse/validator"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

// SynapseValidatorTest implements IntegrationTest
type SynapseValidatorTest struct {
}

// Execute implements IntegrationTest
func (test SynapseValidatorTest) Execute() error {
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
				return fmt.Errorf("invalid validators parameter")
			}
			validatorIndices = append(validatorIndices, uint32(i))
		} else {
			parts := strings.SplitN(s, "-", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid validators parameter")
			}
			first, err := strconv.Atoi(parts[0])
			if err != nil {
				return fmt.Errorf("invalid validators parameter")
			}
			second, err := strconv.Atoi(parts[1])
			if err != nil {
				return fmt.Errorf("invalid validators parameter")
			}
			for i := first; i <= second; i++ {
				validatorIndices = append(validatorIndices, uint32(i))
			}
		}
	}

	logrus.WithField("validators", *validators).Debug("running with validators")

	conn, err := grpc.Dial(*beaconHost, grpc.WithInsecure())
	if err != nil {
		return err
	}

	vm, err := validator.NewManager(conn, validatorIndices, &validator.FakeKeyStore{})
	if err != nil {
		return err
	}

	vm.Start()

	return nil
}
