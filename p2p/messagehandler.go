package p2p

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
)

// writeMessage writes a message and message name to the provided writer.
func writeMessage(message proto.Message, writer *bufio.Writer) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	messageName := proto.MessageName(message)
	nameBytes := []byte(messageName)
	nameLength := len(nameBytes)
	buf := make([]byte, 4)

	// 1 byte for name length + name length + data length
	binary.LittleEndian.PutUint32(buf, uint32(len(data)+1+nameLength))
	_, err = writer.Write(buf)
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte{byte(nameLength)})
	if err != nil {
		return err
	}

	_, err = writer.Write(nameBytes)
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	if err != nil {
		return err
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

// readMessage reads a message name and message from the provided reader.
func readMessage(length uint32, reader *bufio.Reader) (proto.Message, error) {
	nameLengthByte, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	nameLength := uint8(nameLengthByte)
	nameBytes := make([]byte, nameLength)

	_, err = io.ReadFull(reader, nameBytes)
	if err != nil {
		return nil, err
	}
	messageName := string(nameBytes)
	if !strings.HasPrefix(messageName, "pb.") {
		return nil, fmt.Errorf("invalid message name")
	}
	messageType := proto.MessageType(messageName)
	if messageType == nil {
		return nil, fmt.Errorf("could not get message type")
	}

	t := messageType.Elem()
	messagePtr := reflect.New(t)
	message := messagePtr.Interface().(proto.Message)
	if message == nil {
		return nil, fmt.Errorf("could not find message type \"%s\"", messageName)
	}

	data := make([]byte, length-1-uint32(nameLength))
	_, err = io.ReadFull(reader, data)
	if err != nil {
		return nil, err
	}
	err = proto.Unmarshal(data, message)
	if err != nil {
		return nil, err
	}

	return message, nil
}

// processMessages continuously reads from stream and handles any protobuf messages.
func processMessages(stream *bufio.Reader, handler func(message proto.Message) error) error {
	const stateReadHeader = 1
	const stateReadMessage = 2

	headerBuffer := make([]byte, 4)
	state := stateReadHeader
	var messageBuffer []byte
	for {
		switch state {
		case stateReadHeader:
			if _, err := io.ReadFull(stream, headerBuffer); err == nil {
				messageLength := binary.LittleEndian.Uint32(headerBuffer)
				messageBuffer = make([]byte, messageLength)
				state = stateReadMessage
			} else {
				return err
			}

		case stateReadMessage:
			if _, err := io.ReadFull(stream, messageBuffer); err == nil {
				state = stateReadHeader
				message, err := readMessage(uint32(len(messageBuffer)), bufio.NewReader(bytes.NewReader(messageBuffer)))
				if err != nil {
					return err
				}
				err = handler(message)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
}
