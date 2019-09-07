package main

import (
	"github.com/c-bata/go-prompt"
	"github.com/fatih/color"
	"os"
	"strings"
)

func commandCompleter(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "getbalance", Description: "Gets the balance of an address"},
		{Text: "sendtoaddress", Description: "Sends money from one address to another"},
		{Text: "redeem", Description: "Redeems the premine of a certain address"},
		{Text: "exit", Description: "Exits the wallet"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

var output = color.New(color.FgCyan)
var errOut = color.New(color.FgRed, color.Bold)

func exit(b *prompt.Buffer) {
	output.Println("Exiting wallet...")
	os.Exit(0)
}

func getbalance(args []string) {
	if len(args) != 1 {
		errOut.Println("Usage: getbalance <address>")
		return
	}
	output.Printf("Getting balance of %s...\n", args[0])
}

func redeem(args []string) {
	if len(args) != 1 {
		errOut.Println("Usage: redeem <address>")
		return
	}
	output.Printf("Redeeming premine balance of %s...\n", args[0])
}

func exitCommand(args []string) {
	exit(nil)
}

func sendtoaddress(args []string) {
	if len(args) != 3 {
		errOut.Println("Usage: sendtoaddress <amount> <fromaddress> <toaddress>")
		return
	}
	output.Printf("Sending %s PHR from %s to %s...\n", args[0], args[1], args[2])
}

var commandMap = map[string]func(args []string){
	"getbalance":    getbalance,
	"redeem":        redeem,
	"exit":          exitCommand,
	"sendtoaddress": sendtoaddress,
}

func main() {
	for {
		out := prompt.Input("> ", commandCompleter,
			prompt.OptionAddKeyBind(prompt.KeyBind{Key: prompt.ControlC, Fn: exit}),
			prompt.OptionAddKeyBind(prompt.KeyBind{Key: prompt.ControlD, Fn: exit}))

		args := strings.Split(out, " ")

		if len(args) == 0 {
			continue
		}

		comFunc, found := commandMap[args[0]]
		if !found {
			errOut.Printf("invalid command: %s\n", args[0])
			continue
		}

		comFunc(args[1:])
	}
}
