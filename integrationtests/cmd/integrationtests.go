package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/phoreproject/synapse/integrationtests"
)

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

	var entries []integrationtests.Entry
	if len(flagTest) == 0 {
		for _, entry := range integrationtests.EntryList {
			entries = append(entries, entry)
		}
	} else {
		for _, name := range flagTest {
			entries = append(entries, findEntryByName(name))
		}
	}

	errorCount := 0
	for _, entry := range entries {
		fmt.Printf("Running %s\n\n", entry.Name)
		test := entry.Creator()
		service := integrationtests.TestService{
			EntryArgs: entry.EntryArgs,
		}
		err := test.Execute(&service)
		if err != nil {
			errorCount++
			fmt.Println(err)
		}
		fmt.Printf("\n\n")
	}

	fmt.Printf("All done\n")

	os.Exit(errorCount)
}

func findEntryByName(name string) integrationtests.Entry {
	for _, entry := range integrationtests.EntryList {
		if strings.EqualFold(entry.Name, name) {
			return entry
		}
	}

	panic("Can't find test: " + name)
}
