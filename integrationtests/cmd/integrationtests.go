package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/phoreproject/synapse/integrationtests"
	"github.com/phoreproject/synapse/integrationtests/framework"
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

	var entries []testframework.Entry
	if len(flagTest) == 0 {
		for _, entry := range testcase.EntryList {
			entries = append(entries, entry)
		}
	} else {
		for _, name := range flagTest {
			entries = append(entries, findEntryByName(name))
		}
	}

	service := testframework.NewTestService()
	defer service.CleanUp()

	errorCount := 0
	for _, entry := range entries {
		fmt.Printf("Running %s\n\n", entry.Name)
		test := entry.Creator()
		service.BindEntryArgs(entry.EntryArgs)
		err := test.Execute(service)
		if err != nil {
			errorCount++
			fmt.Println(err)
		}
		fmt.Printf("\n\n")
	}

	fmt.Printf("All done\n")

	os.Exit(errorCount)
}

func findEntryByName(name string) testframework.Entry {
	for _, entry := range testcase.EntryList {
		if strings.EqualFold(entry.Name, name) {
			return entry
		}
	}

	panic("Can't find test: " + name)
}
