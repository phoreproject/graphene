package utils

import (
	"encoding/binary"
	"io"
)

// Writer is a writer
type Writer struct {
	writer io.Writer
	endian binary.ByteOrder
	buffer []byte
}

// NewWriterWithEndian creates a writer
func NewWriterWithEndian(writer io.Writer, endian binary.ByteOrder) *Writer {
	return &Writer{
		writer: writer,
		endian: endian,
		buffer: make([]byte, 8),
	}
}

// NewWriter creates a writer
func NewWriter(writer io.Writer) *Writer {
	return NewWriterWithEndian(writer, binary.LittleEndian)
}

// WriteUint8 writes uint8
func (w Writer) WriteUint8(v uint8) {
	buf := w.buffer[0:1]
	buf[0] = v
	w.writer.Write(buf)
}

// WriteUint16 writes uint16
func (w Writer) WriteUint16(v uint16) {
	buf := w.buffer[0:2]
	w.endian.PutUint16(buf, v)
	w.writer.Write(buf)
}

// WriteUint32 writes uint32
func (w Writer) WriteUint32(v uint32) {
	buf := w.buffer[0:4]
	w.endian.PutUint32(buf, v)
	w.writer.Write(buf)
}

// WriteUint64 writes uint64
func (w Writer) WriteUint64(v uint64) {
	buf := w.buffer[0:8]
	w.endian.PutUint64(buf, v)
	w.writer.Write(buf)
}

// WriteBytes writes []byte
func (w Writer) WriteBytes(v []byte) {
	w.writer.Write(v)
}
