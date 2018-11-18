package serialization

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// ReadBytes reads a certain number of bytes guaranteed from
// a reader.
func ReadBytes(r io.Reader, numBytes int) ([]byte, error) {
	out := make([]byte, numBytes)
	n, err := r.Read(out)
	if n != numBytes {
		return nil, errors.New("could not read enough bytes")
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ReadVarInt will read a varint from a reader. If the varint
// is a single byte below 0xfd, the byte is the value. If the first
// byte is 0xfd, we'll return a uint16. If the first byte is 0xfe,
// we'll return a uint32. If the first byte is 0xff, we'll return a
// uint64.
func ReadVarInt(r io.Reader) (uint64, error) {
	i, err := ReadBytes(r, 1)
	if err != nil {
		return 0, err
	}
	iInt := uint8(i[0])
	if iInt < 0xfd {
		return uint64(iInt), nil
	} else if iInt == 0xfd {
		i2, err := ReadBytes(r, 2)
		if err != nil {
			return 0, err
		}
		return uint64(binary.BigEndian.Uint16(i2)), nil
	} else if iInt == 0xfe {
		i2, err := ReadBytes(r, 4)
		if err != nil {
			return 0, err
		}
		return uint64(binary.BigEndian.Uint32(i2)), nil
	}
	i2, err := ReadBytes(r, 8)
	if err != nil {
		return 0, err
	}
	return uint64(binary.BigEndian.Uint64(i2)), nil
}

// WriteVarInt will get the binary representation of a varint.
func WriteVarInt(i uint64) []byte {
	if i < 0xfd {
		return []byte{byte(uint8(i))}
	} else if i < math.MaxUint16 {
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(i))
		return append([]byte{'\xfd'}, b...)
	} else if i < math.MaxUint32 {
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(i))
		return append([]byte{'\xfe'}, b...)
	}
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return append([]byte{'\xff'}, b...)
}

// ReadByteArray reads a byte array from a reader encoded as
// a length as a varint followed by [length]byte
func ReadByteArray(r io.Reader) ([]byte, error) {
	i, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}
	return ReadBytes(r, int(i))
}

// ReadUint32Array reads a variable-length uint32 array
// from reader r.
func ReadUint32Array(r io.Reader) ([]uint32, error) {
	i, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}
	arr := make([]uint32, i)
	for i := range arr {
		b, err := ReadBytes(r, 4)
		if err != nil {
			return nil, err
		}
		arr[i] = binary.BigEndian.Uint32(b)
	}
	return arr, nil
}

// WriteUint32Array gets the binary representation of a uint32 byte
// array.
func WriteUint32Array(toWrite []uint32) []byte {
	lenBytes := WriteVarInt(uint64(len(toWrite)))
	b := lenBytes
	for _, i := range toWrite {
		var intBytes [4]byte
		binary.BigEndian.PutUint32(intBytes[:], i)
		b = append(b, intBytes[:]...)
	}
	return b
}

// WriteByteArray gets the binary representation of a byte array.
func WriteByteArray(toWrite []byte) []byte {
	return append(WriteVarInt(uint64(len(toWrite))), toWrite...)
}
