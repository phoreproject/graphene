package execution

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// ArgumentContext represents a single execution of a shard.
type ArgumentContext interface {
	LoadArgument(argumentNumber int32) []byte
}

// SerializeTransactionWithArguments serializes a transaction with the given arguments.
func SerializeTransactionWithArguments(args ...[]byte) []byte {
	buf := bytes.NewBuffer([]byte{})

	argsLenBuf := make([]byte, binary.MaxVarintLen16)
	binary.PutUvarint(argsLenBuf, uint64(len(args)))
	buf.Write(argsLenBuf)

	for i := range args {
		argLenBuf := make([]byte, binary.MaxVarintLen16)
		binary.PutUvarint(argLenBuf, uint64(len(args[i])))
		buf.Write(argLenBuf)

		buf.Write(args[i])
	}

	return buf.Bytes()
}

// LoadArgumentContextFromTransaction creates an ArgumentContext from a raw transaction.
func LoadArgumentContextFromTransaction(transaction []byte) (IndexedContext, error) {
	// encoding: num arguments + (arglen0 + arg0) + (arglen1 + arg1) + ...
	buf := bytes.NewBuffer(transaction)

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

	return NewIndexedContext(data), nil
}

// EmptyContext is a context that doesn't resolve arguments.
type EmptyContext struct{}

// LoadArgument for empty context doesn't load anything.
func (e EmptyContext) LoadArgument(argumentNumber int32) []byte { return nil }

// IndexedContext is an ArgumentContext that returns the inputted data for the defined arguments and an empty slice otherwise.
type IndexedContext [][]byte

// NewIndexedContext creates a new indexed context.
func NewIndexedContext(data [][]byte) IndexedContext {
	return IndexedContext(data)
}

// LoadArgument for empty context doesn't load anything.
func (i IndexedContext) LoadArgument(argumentNumber int32) []byte {
	if argumentNumber < int32(len(i)) && argumentNumber > 0 {
		return i[argumentNumber]
	}
	return []byte{}
}

// Len gets the number of arguments in the IndexedContext.
func (i IndexedContext) Len() int {
	return len(i)
}

var _ ArgumentContext = EmptyContext{}
var _ ArgumentContext = IndexedContext{}
