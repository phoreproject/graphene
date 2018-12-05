package integrationtests

import (
	"fmt"
)

// SampleTest implements IntegrationTest
type SampleTest struct {
}

// Execute implements IntegrationTest
func (test SampleTest) Execute() error {
	fmt.Printf("This is sample test.")
	return nil
}
