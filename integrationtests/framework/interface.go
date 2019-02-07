package testframework

// EntryArgList is the entry args
type EntryArgList = map[string]interface{}

// Entry is the registery entry
type Entry struct {
	// name is case insensitive
	Name      string
	Creator   func() IntegrationTest
	EntryArgs EntryArgList
}

// IntegrationTest is the interface for all integration tests
// All tests must implement this interface
type IntegrationTest interface {
	// Execute is the main entry of a test.
	// It returns any error if an error happens
	Execute(*TestService) error
}
