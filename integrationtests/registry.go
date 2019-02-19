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
		Name:    "p2p",
		Creator: func() testframework.IntegrationTest { return testcase.P2pTest{} },
	},

	testframework.Entry{
		Name:    "directmessage",
		Creator: func() testframework.IntegrationTest { return testcase.DirectMessageTest{} },
	},

	testframework.Entry{
		Name:    "SynapseP2P",
		Creator: func() testframework.IntegrationTest { return SynapseP2pTest{} },
		EntryArgs: testframework.EntryArgList{
			"listen":             flag.String("listen", "/ip4/0.0.0.0/tcp/11781", "specifies the address to listen on"),
			"initialConnections": flag.String("connect", "", "comma separated multiaddrs"),
			"rpcConnect":         flag.String("rpclisten", "127.0.0.1:11783", "host and port for RPC server to listen on"),
		},
	},
}
