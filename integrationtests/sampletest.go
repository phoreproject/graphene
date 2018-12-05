package integrationtests

import (
	"flag"
	"fmt"
)

// SampleTest implements IntegrationTest
type SampleTest struct {
}

// GetCommandLineFlag gets the flag
func GetCommandLineFlag(name string, defaultValue string) string {
	result := defaultValue
	flag.VisitAll(func(f *flag.Flag) {
		if f.Name == name {
			result = f.Value.String()
		}
	})
	return result
}

// Execute implements IntegrationTest
func (test SampleTest) Execute(service *TestService) error {
	arg := service.GetArgString("arg", "n/a")
	fmt.Printf("This is sample test, arg = " + arg)
	return nil
}
