package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/phoreproject/synapse/integrationtests"
)

type entry struct {
	// name is case insensitive
	name    string
	creator func() integrationtests.IntegrationTest
}

var entryList = []entry{
	entry{name: "sample", creator: func() integrationtests.IntegrationTest { return integrationtests.SampleTest{} }},
	entry{name: "SynapseP2p", creator: func() integrationtests.IntegrationTest { return integrationtests.SynapseP2pTest{} }},
}

type multipleFlags []string

func (i *multipleFlags) String() string {
	return "Multiple options"
}

func (i *multipleFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var flagTest multipleFlags
	flag.Var(&flagTest, "test", "Specify the test name, can't multiple.")
	flag.Parse()

	var entries []entry
	if len(flagTest) == 0 {
		for _, entry := range entryList {
			entries = append(entries, entry)
		}
	} else {
		for _, name := range flagTest {
			entries = append(entries, findEntryByName(name))
		}
	}

	errorCount := 0
	for _, entry := range entries {
		fmt.Printf("Running %s\n\n", entry.name)
		test := entry.creator()
		err := test.Execute()
		if err != nil {
			errorCount++
			fmt.Println(err)
		}
		fmt.Printf("\n\n")
	}

	fmt.Printf("All done\n")

	os.Exit(errorCount)
}

func findEntryByName(name string) entry {
	for _, entry := range entryList {
		if strings.EqualFold(entry.name, name) {
			return entry
		}
	}

	panic("Can't find test: " + name)
}
