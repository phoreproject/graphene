package rpc

import (
	"net"
	"net/http"
	"net/rpc"

	logger "github.com/inconshreveable/log15"
)

//Holds arguments to be passed to service Arith in RPC call
type Args struct {
	A, B int
}

//Representss service Arith with method Multiply
type Arith int

//Result of RPC call is of this type
type Result int

//This procedure is invoked by rpc and calls rpcexample.Multiply which stores product of args.A and args.B in result pointer
func (t *Arith) Multiply(args Args, result *Result) error {
	return Multiply(args, result)
}

//stores product of args.A and args.B in result pointer
func Multiply(args Args, result *Result) error {
	logger.Debug("multiplying", "a", args.A, "b", args.B)
	*result = Result(args.A * args.B)
	return nil
}

func Serve(hostPort string) error {
	//register Arith object as a service
	arith := new(Arith)
	err := rpc.Register(arith)
	if err != nil {
		return err
	}
	rpc.HandleHTTP()
	//start listening for messages on port 1234
	l, e := net.Listen("tcp", hostPort)
	if e != nil {
		return err
	}
	logger.Debug("serving RPC handler")
	err = http.Serve(l, nil)
	if err != nil {
		return err
	}
	return nil
}
