package integrationtests

import (
	"flag"
)

// EntryList is the registery entry list
var EntryList = []Entry{
	Entry{
		Name:      "sample",
		Creator:   func() IntegrationTest { return SampleTest{} },
		EntryArgs: EntryArgList{"arg": flag.String("arg", "n/a", "The sample arg")},
	},
	Entry{
		Name:    "SynapseP2p",
		Creator: func() IntegrationTest { return SynapseP2pTest{} },
		EntryArgs: EntryArgList{
			"listen":             flag.String("listen", "/ip4/0.0.0.0/tcp/11781", "specifies the address to listen on"),
			"initialConnections": flag.String("connect", "", "comma separated multiaddrs"),
			"rpcConnect":         flag.String("rpclisten", "127.0.0.1:11783", "host and port for RPC server to listen on"),
		},
	},
}
