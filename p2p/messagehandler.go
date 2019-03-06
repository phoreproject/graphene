package p2p

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"reflect"

	proto "github.com/golang/protobuf/proto"
	inet "github.com/libp2p/go-libp2p-net"
	logger "github.com/sirupsen/logrus"
)

func writeMessage(message proto.Message, writer *bufio.Writer) error {
	data, err := proto.Marshal(message)
	if err != nil {
		logger.WithField("Function", "writeMessage").Warn(err)
		return err
	}

	logger.WithField("Function", "writeMessage").Infof("Sending %s", proto.MessageName(message))
	messageName := proto.MessageName(message)
	nameBytes := []byte(messageName)
	nameLength := len(nameBytes)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(len(data)+1+nameLength))
	written, err := writer.Write(buf)
	if err != nil {
		logger.WithField("Function", "writeMessage").Warn(err)
		return err
	}
	if written != len(buf) {
		logger.WithField("Function", "writeMessage").Warn("written < len(buf)")
		return nil
	}

	writer.WriteByte(byte(nameLength))

	written, err = writer.Write(nameBytes)
	if err != nil {
		logger.WithField("Function", "writeMessage").Warn(err)
		return err
	}
	if written != len(nameBytes) {
		logger.WithField("Function", "writeMessage").Warn("written < len(nameBytes)")
		return nil
	}

	written, err = writer.Write(data)
	if err != nil {
		logger.WithField("Function", "writeMessage").Warn(err)
		return err
	}
	if written != len(data) {
		logger.WithField("Function", "writeMessage").Warn("written < len(data)")
		return nil
	}

	writer.Flush()

	proto.Unmarshal(data, message)

	logger.WithField("Function", "writeMessage").Debugf("Written message %s %v Length: %d", messageName, reflect.TypeOf(message), len(data))

	return nil
}

func readMessage(length uint32, reader *bufio.Reader) (proto.Message, error) {
	nameLength, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	nameBytes := make([]byte, nameLength)
	reader.Read(nameBytes)
	messageName := string(nameBytes)
	messageType := proto.MessageType(messageName)
	message := reflect.New(messageType.Elem()).Interface().(proto.Message)
	if message == nil {
		logger.WithField("Function", "readMessage").Warnf("Message is nil! %s %v", messageName, messageType)
	}

	data := make([]byte, length-1-uint32(nameLength))
	reader.Read(data)
	proto.Unmarshal(data, message)

	return message, nil
}

func processMessages(stream inet.Stream, handler func(message proto.Message)) {
	const stateReadHeader = 1
	const stateReadMessage = 2

	streamReader := NewStreamReader(stream)
	headerBuffer := make([]byte, 4)
	state := stateReadHeader
	var messageBuffer []byte
	for {
		switch state {
		case stateReadHeader:
			if streamReader.Read(headerBuffer) {
				messageLength := binary.LittleEndian.Uint32(headerBuffer)
				messageBuffer = make([]byte, messageLength)
				state = stateReadMessage
			}
			break

		case stateReadMessage:
			if streamReader.Read(messageBuffer) {
				state = stateReadHeader
				message, _ := readMessage(uint32(len(messageBuffer)), bufio.NewReader(bytes.NewReader(messageBuffer)))
				handler(message)
			}
			break
		}
	}
}
