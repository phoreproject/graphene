package cmd

import (
	"flag"
	"github.com/c-bata/go-prompt"
	"github.com/fatih/color"
	"github.com/phoreproject/synapse/wallet/rpc"
	"google.golang.org/grpc"
	"os"
	"strings"
)

func commandCompleter(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "getbalance", Description: "Gets the balance of an address"},
		{Text: "sendtoaddress", Description: "Sends money from one address to another"},
		{Text: "redeem", Description: "Redeems the premine of a certain address"},
		{Text: "exit", Description: "Exits the wallet"},
		{Text: "getnewaddress", Description: "Generates a new address"},
		{Text: "importprivkey", Description: "Imports a private key"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

var output = color.New(color.FgCyan)

var errOut = color.New(color.FgRed, color.Bold)

func exit(b *prompt.Buffer) {
	os.Exit(0)
}

func main() {
	shardHost := flag.String("shardhost", "localhost:11783", "host of shard module to connect to")

	flag.Parse()

	c, err := grpc.Dial(*shardHost, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	walletRPC := rpc.NewWalletRPC(c, *output, *errOut)

	var commandMap = map[string]func(args []string){
		"getbalance":    walletRPC.GetBalance,
		"redeem":        walletRPC.Redeem,
		"exit":          walletRPC.Exit,
		"sendtoaddress": walletRPC.SendToAddress,
		"getnewaddress": walletRPC.GetNewAddress,
		"importprivkey": walletRPC.ImportPrivKey,
	}

	go func() {
		<-walletRPC.ExitChan
		exit(nil)
	}()

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
			_, _ = errOut.Printf("invalid command: %s\n", args[0])
			continue
		}

		comFunc(args[1:])
	}
}
