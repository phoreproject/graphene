package integrationtests

// IntegrationTest is the interface for all integration tests
// All tests must implement this interface
type IntegrationTest interface {
	// Execute is the main entry of a test.
	// It returns any error if an error happens
	Execute() error
}
