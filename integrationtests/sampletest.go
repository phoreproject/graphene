package testcase

import (
	"fmt"

	"github.com/phoreproject/synapse/integrationtests/framework"
)

// SampleTest implements IntegrationTest
type SampleTest struct {
}

// Execute implements IntegrationTest
func (test SampleTest) Execute(service *testframework.TestService) error {
	arg := service.GetArgString("arg", "n/a")
	fmt.Printf("This is sample test, arg = " + arg)
	return nil
}
