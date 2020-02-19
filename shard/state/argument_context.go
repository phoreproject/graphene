package state

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// ArgumentContext represents a single execution of a shard.
type ArgumentContext interface {
	LoadArgument(argumentNumber int32, argLen int32) ([]byte, error)
	GetFunction() string
}

// SerializeTransactionWithArguments serializes a transaction with the given arguments.
func SerializeTransactionWithArguments(fnName string, args ...[]byte) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	if len(fnName) > 255 {
		return nil, errors.New("function name too long")
	}

	buf.Write([]byte{byte(len(fnName))})
	buf.Write([]byte(fnName))

	argsLenBuf := make([]byte, binary.MaxVarintLen16)
	n := binary.PutUvarint(argsLenBuf, uint64(len(args)))
	buf.Write(argsLenBuf[:n])

	for i := range args {
		argLenBuf := make([]byte, binary.MaxVarintLen16)
		n := binary.PutUvarint(argLenBuf, uint64(len(args[i])))
		buf.Write(argLenBuf[:n])

		buf.Write(args[i])
	}

	return buf.Bytes(), nil
}

// LoadArgumentContextFromTransaction creates an ArgumentContext from a raw transaction.
func LoadArgumentContextFromTransaction(transaction []byte) (*IndexedContext, error) {
	// encoding: num arguments + (arglen0 + arg0) + (arglen1 + arg1) + ...
	buf := bytes.NewBuffer(transaction)

	lenFnNameByte, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	fnName := make([]byte, lenFnNameByte)
	n, err := buf.Read(fnName)
	if err != nil {
		return nil, err
	}

	if n != int(lenFnNameByte) {
		return nil, fmt.Errorf("malformatted transaction")
	}

	numArgs, err := binary.ReadUvarint(buf)
	if err != nil {
		return nil, err
	}

	data := make([][]byte, numArgs)

	for i := 0; i < int(numArgs); i++ {
		lenArg, err := binary.ReadUvarint(buf)
		if err != nil {
			return nil, err
		}

		argumentBuf := make([]byte, lenArg)

		n, err := buf.Read(argumentBuf)
		if err != nil {
			return nil, err
		}

		if n != int(lenArg) {
			return nil, fmt.Errorf("malformatted transaction")
		}

		data[i] = argumentBuf
	}

	return NewIndexedContext(string(fnName), data), nil
}

// EmptyContext is a context that doesn't resolve arguments.
type EmptyContext struct {
	function string
}

// NewEmptyContext creates a new empty context with no arguments and a function name.
func NewEmptyContext(fnName string) EmptyContext {
	return EmptyContext{
		function: fnName,
	}
}

// GetFunction gets the function associated with the EmptyContext.
func (e EmptyContext) GetFunction() string {
	return e.function
}

// LoadArgument for empty context doesn't load anything.
func (e EmptyContext) LoadArgument(argumentNumber int32, argLen int32) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// IndexedContext is an ArgumentContext that returns the inputted data for the defined arguments and an empty slice otherwise.
type IndexedContext struct {
	args     [][]byte
	function string
}

// NewIndexedContext creates a new indexed context.
func NewIndexedContext(fnName string, data [][]byte) *IndexedContext {
	return &IndexedContext{
		args:     data,
		function: fnName,
	}
}

// LoadArgument for empty context doesn't load anything.
func (i *IndexedContext) LoadArgument(argumentNumber int32, argLen int32) ([]byte, error) {
	if argumentNumber < int32(len(i.args)) && argumentNumber >= 0 {
		if len(i.args[argumentNumber]) == int(argLen) {
			return i.args[argumentNumber], nil
		}

		return nil, fmt.Errorf("invalid argument length: %d (expected: %d)", argLen, len(i.args[argumentNumber]))
	}

	return nil, fmt.Errorf("invalid argument number: %d (max: %d)", argumentNumber, len(i.args))
}

// GetFunction gets the function associated with the IndexedContext.
func (i *IndexedContext) GetFunction() string {
	return i.function
}

// Len gets the number of arguments in the IndexedContext.
func (i *IndexedContext) Len() int {
	return len(i.args)
}

var _ ArgumentContext = EmptyContext{}
var _ ArgumentContext = &IndexedContext{}
