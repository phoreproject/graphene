package p2p

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"

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

	err = proto.Unmarshal(data, message)
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
	nameByte := make([]byte, 1)
	_, err := reader.Read(nameByte)
	if err != nil {
		return nil, err
	}
	nameLength := uint8(nameByte[0])
	nameBytes := make([]byte, nameLength)

	_, err = reader.Read(nameBytes)
	messageName := string(nameBytes)
	messageType := proto.MessageType(messageName)
	message := reflect.New(messageType.Elem()).Interface().(proto.Message)
	if message == nil {
		return nil, fmt.Errorf("could not find message type \"%s\"", messageName)
	}

	data := make([]byte, length-1-uint32(nameLength))
	_, err = reader.Read(data)
	if err != nil {
		return nil, err
	}
	err = proto.Unmarshal(data, message)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func processMessages(stream *bufio.Reader, handler func(message proto.Message) error) error {
	const stateReadHeader = 1
	const stateReadMessage = 2

	headerBuffer := make([]byte, 4)
	state := stateReadHeader
	var messageBuffer []byte
	for {
		switch state {
		case stateReadHeader:
			if _, err := stream.Read(headerBuffer); err == nil {
				messageLength := binary.LittleEndian.Uint32(headerBuffer)
				messageBuffer = make([]byte, messageLength)
				state = stateReadMessage
			} else {
				return err
			}
			break

		case stateReadMessage:
			if _, err := stream.Read(messageBuffer); err == nil {
				state = stateReadHeader
				message, _ := readMessage(uint32(len(messageBuffer)), bufio.NewReader(bytes.NewReader(messageBuffer)))
				err := handler(message)
				if err != nil {
					return err
				}
			} else {
				return err
			}
			break
		}
	}
}

// MessageToBytes converts a message to byte array
func MessageToBytes(message proto.Message) ([]byte, error) {
	var buf bytes.Buffer
	err := writeMessage(message, bufio.NewWriter(&buf))
	if err != nil {
		return nil, err
	}

	messageLen := buf.Len()
	result := make([]byte, messageLen)
	copy(result, buf.Bytes())
	return result, nil
}

// BytesToMessage converts byte array to message
func BytesToMessage(messageBuffer []byte) (proto.Message, error) {
	message, err := readMessage(uint32(len(messageBuffer)), bufio.NewReader(bytes.NewReader(messageBuffer)))
	return message, err
}
