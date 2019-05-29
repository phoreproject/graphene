package testcase

import (
	testframework "github.com/phoreproject/synapse/integrationtests/framework"
)

// EntryList is the registery entry list
var EntryList = []testframework.Entry{
	{
		Name: "Beacon-Validator Interaction",
		Creator: func() testframework.IntegrationTest {
			return &ValidateTest{}
		},
	},
}
