package rpc

import (
	"errors"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/wallet"
	"github.com/phoreproject/synapse/wallet/address"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

// WalletRPC handles wallet RPC commands.
type WalletRPC struct {
	w        wallet.Wallet
	ExitChan chan struct{}
}

// NewWalletCMD creates a new WalletCMD for handling wallet CMD commands.
func NewWalletRPC(relayerConn *grpc.ClientConn) *WalletRPC {
	return &WalletRPC{
		w:        wallet.NewWallet(relayerConn),
		ExitChan: make(chan struct{}),
	}
}

// WaitForExit returns a channel that resolves when an exit is requested.
func (w *WalletRPC) WaitForExit() chan struct{} {
	return w.ExitChan
}

// Exit exits the wallet.
func (w *WalletRPC) Exit(args []string) {
	w.ExitChan <- struct{}{}
}

type server struct {
	rpc *WalletRPC
}

// GetBalance gets the balance of an address
func (s *server) GetBalance(ctx context.Context, request *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	if !address.ValidateAddress(request.Address) {
		return &pb.GetBalanceResponse{}, errors.New("provided address is invalid")
	}

	balance, err := s.rpc.w.GetBalance(address.Address(request.Address))
	if err != nil {
		return &pb.GetBalanceResponse{}, err
	}

	return &pb.GetBalanceResponse{
		Balance: balance,
	}, err
}

// SendToAddress sends money to a certain address.
func (s *server) SendToAddress(ctx context.Context, request *pb.SendToAddressRequest) (*pb.SendToAddressResponse, error) {
	if !address.ValidateAddress(request.From) {
		return &pb.SendToAddressResponse{
			Success: false,
		}, errors.New("'From' address is invalid")
	}

	if !address.ValidateAddress(request.To) {
		return &pb.SendToAddressResponse{
			Success: false,
		}, errors.New("'To' address is invalid")
	}

	fromAddr := address.Address(request.From)
	toAddr := address.Address(request.To)

	err := s.rpc.w.SendToAddress(fromAddr, toAddr, request.Value)
	if err != nil {
		return &pb.SendToAddressResponse{
			Success: false,
		}, err
	}

	return &pb.SendToAddressResponse{
		Success: true,
	}, nil
}

// Redeem redeems the premine of an address.
func (s *server) Redeem(ctx context.Context, request *pb.RedeemRequest) (*pb.RedeemResponse, error) {
	if !address.ValidateAddress(request.Address) {
		return &pb.RedeemResponse{Success: false}, errors.New("address is not valid")
	}

	err := s.rpc.w.RedeemPremine(address.Address(request.Address))
	if err != nil {
		return &pb.RedeemResponse{Success: false}, err
	}

	return &pb.RedeemResponse{Success: true}, nil
}

// GetNewAddress gets a new address.
func (s *server) GetNewAddress(ctx context.Context, request *pb.GetNewAddressRequest) (*pb.GetNewAddressResponse, error) {
	addr, err := s.rpc.w.GetNewAddress(uint32(request.ShardID))
	if err != nil {
		return &pb.GetNewAddressResponse{Address: string(addr)}, nil
	}

	return &pb.GetNewAddressResponse{Address: string(addr)}, nil
}

// ImportPrivKey imports a private key.
func (s *server) ImportPrivateKey(ctx context.Context, request *pb.ImportPrivateKeyRequest) (*pb.ImportPrivateKeyResponse, error) {
	addr := s.rpc.w.ImportPrivKey(request.PrivateKey, uint32(request.ShardID))

	return &pb.ImportPrivateKeyResponse{Address: string(addr)}, nil
}

// Serve serves the RPC server
func Serve(proto string, listenAddr string, rpc *WalletRPC) error {
	lis, err := net.Listen(proto, listenAddr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterWalletRPCServer(s, &server{rpc})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	return err
}
