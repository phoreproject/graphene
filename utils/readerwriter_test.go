package utils

import (
	"bytes"
	"encoding/binary"
	"testing"
)

type readerWriter struct {
	reader *Reader
	writer *Writer
}

func newReaderWriter(readerEndian, writerEndian binary.ByteOrder) readerWriter {
	buffer := &bytes.Buffer{}
	rw := readerWriter{}
	if readerEndian == nil {
		rw.reader = NewReader(buffer)
	} else {
		rw.reader = NewReaderWithEndian(buffer, readerEndian)
	}
	if writerEndian == nil {
		rw.writer = NewWriter(buffer)
	} else {
		rw.writer = NewWriterWithEndian(buffer, writerEndian)
	}
	return rw
}

func TestReaderWriter_defaultEndian(t *testing.T) {
	rw := newReaderWriter(nil, nil)
	rw.writer.WriteUint32(6)
	rw.writer.WriteUint64(19)
	rw.writer.WriteUint16(200)
	rw.writer.WriteUint8(50)

	ui32, err := rw.reader.ReadUint32()
	if err != nil {
		t.Fatal(err)
	}
	if ui32 != 6 {
		t.Error("Error ReadUint32")
	}

	ui64, err := rw.reader.ReadUint64()
	if err != nil {
		t.Fatal(err)
	}
	if ui64 != 19 {
		t.Error("Error ReadUint64")
	}

	ui16, err := rw.reader.ReadUint16()
	if err != nil {
		t.Fatal(err)
	}
	if ui16 != 200 {
		t.Error("Error ReadUint16")
	}

	ui8, err := rw.reader.ReadUint8()
	if err != nil {
		t.Fatal(err)
	}
	if ui8 != 50 {
		t.Error("Error ReadUint8")
	}
}

func TestReaderWriter_littleEndian(t *testing.T) {
	rw := newReaderWriter(binary.LittleEndian, binary.LittleEndian)
	rw.writer.WriteUint32(6)
	rw.writer.WriteUint64(19)
	rw.writer.WriteUint16(200)
	rw.writer.WriteUint8(50)

	ui32, err := rw.reader.ReadUint32()
	if err != nil {
		t.Fatal(err)
	}
	if ui32 != 6 {
		t.Error("Error ReadUint32")
	}

	ui64, err := rw.reader.ReadUint64()
	if err != nil {
		t.Fatal(err)
	}
	if ui64 != 19 {
		t.Error("Error ReadUint64")
	}

	ui16, err := rw.reader.ReadUint16()
	if err != nil {
		t.Fatal(err)
	}
	if ui16 != 200 {
		t.Error("Error ReadUint16")
	}

	ui8, err := rw.reader.ReadUint8()
	if err != nil {
		t.Fatal(err)
	}
	if ui8 != 50 {
		t.Error("Error ReadUint8")
	}
}

func TestReaderWriter_bigEndian(t *testing.T) {
	rw := newReaderWriter(binary.BigEndian, binary.BigEndian)
	rw.writer.WriteUint32(6)
	rw.writer.WriteUint64(19)
	rw.writer.WriteUint16(200)
	rw.writer.WriteUint8(50)

	ui32, err := rw.reader.ReadUint32()
	if err != nil {
		t.Fatal(err)
	}
	if ui32 != 6 {
		t.Error("Error ReadUint32")
	}

	ui64, err := rw.reader.ReadUint64()
	if err != nil {
		t.Fatal(err)
	}
	if ui64 != 19 {
		t.Error("Error ReadUint64")
	}

	ui16, err := rw.reader.ReadUint16()
	if err != nil {
		t.Fatal(err)
	}
	if ui16 != 200 {
		t.Error("Error ReadUint16")
	}

	ui8, err := rw.reader.ReadUint8()
	if err != nil {
		t.Fatal(err)
	}
	if ui8 != 50 {
		t.Error("Error ReadUint8")
	}
}

func TestReaderWriter_wrongEndian(t *testing.T) {
	rw := newReaderWriter(binary.LittleEndian, binary.BigEndian)
	rw.writer.WriteUint32(6)
	rw.writer.WriteUint64(19)
	rw.writer.WriteUint16(200)
	rw.writer.WriteUint8(50)

	ui32, err := rw.reader.ReadUint32()
	if err != nil {
		t.Fatal(err)
	}
	if ui32 == 6 {
		t.Error("Error ReadUint32")
	}

	ui64, err := rw.reader.ReadUint64()
	if err != nil {
		t.Fatal(err)
	}
	if ui64 == 19 {
		t.Error("Error ReadUint64")
	}

	ui16, err := rw.reader.ReadUint16()
	if err != nil {
		t.Fatal(err)
	}
	if ui16 == 200 {
		t.Error("Error ReadUint16")
	}

	ui8, err := rw.reader.ReadUint8()
	if err != nil {
		t.Fatal(err)
	}
	if ui8 != 50 { // this should be 50
		t.Error("Error ReadUint8")
	}
}
