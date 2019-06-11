package utils

import (
	"encoding/binary"
	"io"
)

// Reader is a reader
type Reader struct {
	reader io.Reader
	endian binary.ByteOrder
	buffer []byte
}

// NewReaderWithEndian creates a reader
func NewReaderWithEndian(reader io.Reader, endian binary.ByteOrder) *Reader {
	return &Reader{
		reader: reader,
		endian: endian,
		buffer: make([]byte, 8),
	}
}

// NewReader creates a reader
func NewReader(reader io.Reader) *Reader {
	return NewReaderWithEndian(reader, binary.LittleEndian)
}

// ReadUint8 reads uint8
func (r Reader) ReadUint8() (uint8, error) {
	buf := r.buffer[0:1]
	_, err := r.reader.Read(buf)
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

// ReadUint16 reads uint16
func (r Reader) ReadUint16() (uint16, error) {
	buf := r.buffer[0:2]
	_, err := r.reader.Read(buf)
	if err != nil {
		return 0, err
	}
	return r.endian.Uint16(buf), nil
}

// ReadUint32 reads uint32
func (r Reader) ReadUint32() (uint32, error) {
	buf := r.buffer[0:4]
	_, err := r.reader.Read(buf)
	if err != nil {
		return 0, err
	}
	return r.endian.Uint32(buf), nil
}

// ReadUint64 reads uint64
func (r Reader) ReadUint64() (uint64, error) {
	buf := r.buffer[0:8]
	_, err := r.reader.Read(buf)
	if err != nil {
		return 0, err
	}
	return r.endian.Uint64(buf), nil
}

// ReadBytes reads []byte
func (r Reader) ReadBytes(p []byte) (int, error) {
	return r.reader.Read(p)
}
