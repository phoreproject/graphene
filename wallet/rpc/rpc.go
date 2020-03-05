package rpc

import (
	"encoding/hex"
	"github.com/fatih/color"
	"github.com/phoreproject/synapse/wallet"
	"github.com/phoreproject/synapse/wallet/address"
	"google.golang.org/grpc"
	"strconv"
)

// WalletRPC handles wallet RPC commands.
type WalletRPC struct {
	w        wallet.Wallet
	ExitChan chan struct{}
	out      color.Color
	errOut   color.Color
}

// NewWalletRPC creates a new WalletRPC for handling wallet RPC commands.
func NewWalletRPC(relayerConn *grpc.ClientConn, out color.Color, errOut color.Color) *WalletRPC {
	return &WalletRPC{
		w:        wallet.NewWallet(relayerConn),
		ExitChan: make(chan struct{}),
		out:      out,
		errOut:   errOut,
	}
}

func (w *WalletRPC) println(a ...interface{}) {
	_, _ = w.out.Println(a...)
}

func (w *WalletRPC) printf(f string, a ...interface{}) {
	_, _ = w.out.Printf(f, a...)
}

func (w *WalletRPC) errln(a ...interface{}) {
	_, _ = w.errOut.Println(a...)
}

func (w *WalletRPC) errf(f string, a ...interface{}) {
	_, _ = w.errOut.Printf(f, a...)
}

// WaitForExit returns a channel that resolves when an exit is requested.
func (w *WalletRPC) WaitForExit() chan struct{} {
	return w.ExitChan
}

// Exit exits the wallet.
func (w *WalletRPC) Exit(args []string) {
	w.println("Exiting wallet...")
	w.ExitChan <- struct{}{}
}

// GetBalance gets the balance of an address
func (w *WalletRPC) GetBalance(args []string) {
	if len(args) != 1 {
		w.errln("Usage: getbalance <address>")
		return
	}

	if !address.ValidateAddress(args[0]) {
		w.errf("Invalid address: %s\n", args[0])
		return
	}

	addr := address.Address(args[0])

	w.printf("Getting balance of %s...\n", addr)

	bal, err := w.w.GetBalance(addr)
	if err != nil {
		w.errf("Error getting balance: %s\n", err)
		return
	}

	w.printf("Balance of %s is %d\n", addr, bal)
}

// Redeem redeems the premine of an address.
func (w *WalletRPC) Redeem(args []string) {
	if len(args) != 1 {
		w.errln("Usage: redeem <address>")
		return
	}

	if !address.ValidateAddress(args[0]) {
		w.errf("Invalid address: %s\n", args[0])
		return
	}

	addr := address.Address(args[0])

	w.printf("Redeeming premine balance of %s...\n", addr)

	err := w.w.RedeemPremine(addr)
	if err != nil {
		w.errf("Error redeeming premine: %s\n", err)
		return
	}

	w.println("Successfully redeemed premine.")
}

// SendToAddress sends money to a certain address.
func (w *WalletRPC) SendToAddress(args []string) {
	if len(args) != 3 {
		w.errln("Usage: sendtoaddress <amount> <fromaddress> <toaddress>")
		return
	}

	amount, err := strconv.Atoi(args[0])
	if err != nil {
		w.errf("Error parsing amount: %s", args[0])
	}

	if !address.ValidateAddress(args[1]) {
		w.errf("Invalid address: %s\n", args[1])
		return
	}

	if !address.ValidateAddress(args[2]) {
		w.errf("Invalid address: %s\n", args[2])
		return
	}

	fromAddr := address.Address(args[1])
	toAddr := address.Address(args[2])

	w.printf("Sending %d PHR from %s to %s...\n", amount, fromAddr, toAddr)

	err = w.w.SendToAddress(fromAddr, toAddr, uint64(amount))
	if err != nil {
		w.errf("Error sending: %s\n", err)
		return
	}

	w.printf("Sent %d PHR from %s to %s.\n", amount, fromAddr, toAddr)
}

// GetNewAddress gets a new address.
func (w *WalletRPC) GetNewAddress(args []string) {
	if len(args) != 0 {
		w.errln("Usage: getnewaddress")
		return
	}

	addr, err := w.w.GetNewAddress(0)
	if err != nil {
		w.errf("Error generating new address: %s\n", err)
	}

	w.printf("Generated new address: %s\n", addr)
}

// ImportPrivKey imports a private key.
func (w *WalletRPC) ImportPrivKey(args []string) {
	if len(args) != 1 {
		w.errln("Usage: importprivkey <keyhex>")
		return
	}

	privBytes, err := hex.DecodeString(args[0])
	if err != nil {
		w.errf("Error parsing private key: %s\n", err)
	}

	addr := w.w.ImportPrivKey(privBytes, 0)

	w.printf("Imported new address: %s\n", addr)
}
