package p2p

import (
	"bufio"

	inet "github.com/libp2p/go-libp2p-net"
)

// StreamReader is a stream reader
type StreamReader struct {
	stream inet.Stream
	buffer []byte
	reader *bufio.Reader
}

// NewStreamReader creates a StreamReader
func NewStreamReader(stream inet.Stream) StreamReader {
	return StreamReader{
		stream: stream,
		buffer: make([]byte, 0),
		reader: bufio.NewReader(stream),
	}
}

// Read reads data
func (streamReader *StreamReader) Read(buf []byte) bool {
	if len(streamReader.buffer) < len(buf) {
		bytesToRead := len(buf) - len(streamReader.buffer)
		tempBuffer := make([]byte, bytesToRead)
		readBytes, err := streamReader.reader.Read(tempBuffer)
		if err == nil {
			streamReader.buffer = append(streamReader.buffer, tempBuffer[:readBytes]...)
		}
	}

	if len(streamReader.buffer) < len(buf) {
		return false
	}

	copy(buf, streamReader.buffer[:len(buf)])
	temp := streamReader.buffer
	streamReader.buffer = make([]byte, 0)
	streamReader.buffer = append(streamReader.buffer, temp[len(buf):]...)

	return true
}
