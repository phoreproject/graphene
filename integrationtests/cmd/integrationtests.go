package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/phoreproject/synapse/integrationtests"
	"github.com/phoreproject/synapse/integrationtests/framework"
	logger "github.com/sirupsen/logrus"
)

type multipleFlags []string

func (i *multipleFlags) String() string {
	return "Multiple options"
}

// Set sets a value for the flags
func (i *multipleFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	logger.SetLevel(logger.TraceLevel)

	var flagTest multipleFlags
	flag.Var(&flagTest, "test", "Specify the test name, can't multiple.")
	flag.Parse()

	var entries []testframework.Entry
	if len(flagTest) == 0 {
		entries = append(entries, testcase.EntryList...)
	} else {
		for _, name := range flagTest {
			entries = append(entries, findEntryByName(name))
		}
	}

	service := testframework.NewTestService()

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

	err := service.CleanUp()
	if err != nil {
		panic(err)
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
