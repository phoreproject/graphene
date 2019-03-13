package testcase

import (
	"flag"

	"github.com/phoreproject/synapse/integrationtests/framework"
	"github.com/phoreproject/synapse/integrationtests/p2p"
)

// EntryList is the registery entry list
var EntryList = []testframework.Entry{
	testframework.Entry{
		Name:      "sample",
		Creator:   func() testframework.IntegrationTest { return SampleTest{} },
		EntryArgs: testframework.EntryArgList{"arg": flag.String("arg", "n/a", "The sample arg")},
	},

	testframework.Entry{
		Name:    "p2papp",
		Creator: func() testframework.IntegrationTest { return p2p.TestCase{} },
	},

	testframework.Entry{
		Name:    "p2pbeacon",
		Creator: func() testframework.IntegrationTest { return p2p.TestCaseP2PBeacon{} },
	},
}
