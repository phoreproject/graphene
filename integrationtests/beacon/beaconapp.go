package beaconapp

import (
	"github.com/phoreproject/synapse/beacon/app"
	"github.com/phoreproject/synapse/integrationtests/framework"
)

// BeaconAppTest implements IntegrationTest
type BeaconAppTest struct {
}

// Execute implements IntegrationTest
func (test BeaconAppTest) Execute(service *testframework.TestService) error {
	app := app.GetApp()
	app.Run()

	return nil
}
