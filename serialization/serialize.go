package serialization

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"math/big"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
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
	} else if iInt == 0xfd {
		i2, err := ReadBytes(r, 4)
		if err != nil {
			return 0, err
		}
		return uint64(binary.BigEndian.Uint32(i2)), nil
	} else {
		i2, err := ReadBytes(r, 8)
		if err != nil {
			return 0, err
		}
		return uint64(binary.BigEndian.Uint64(i2)), nil
	}
}

// WriteVarInt will get the binary representation of a varint.
func WriteVarInt(i uint64) []byte {
	if i < 0xfd {
		return []byte{byte(uint8(i))}
	} else {
		if i < math.MaxUint16 {
			var b []byte
			binary.BigEndian.PutUint16(b, uint16(i))
			return append([]byte{'\xfd'}, b...)
		} else if i < math.MaxUint32 {
			var b []byte
			binary.BigEndian.PutUint32(b, uint32(i))
			return append([]byte{'\xfe'}, b...)
		} else {
			var b []byte
			binary.BigEndian.PutUint64(b, uint64(i))
			return append([]byte{'\xff'}, b...)
		}
	}
}

// ReadByteArray reads a byte array from a reader encoded as
// a length as a varint followed by [length]byte
func ReadByteArray(r io.Reader) ([]byte, error) {
	i, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}
	if i > math.MaxUint32 {
		return nil, errors.New("invalid length of byte array (greater than max uint32)")
	}
	return ReadBytes(r, int(i))
}

// WriteByteArray gets the binary representation of a byte array.
func WriteByteArray(toWrite []byte) []byte {
	return append(WriteVarInt(uint64(len(toWrite))), toWrite...)
}

// ReadHash will read a hash from the reader provided.
func ReadHash(r io.Reader) (*chainhash.Hash, error) {
	b, err := ReadBytes(r, chainhash.HashSize)
	if err != nil {
		return nil, err
	}
	return chainhash.NewHash(b)
}

// ReadAddress will read an address from the reader.
func ReadAddress(r io.Reader) (*Address, error) {
	b, err := ReadBytes(r, 20)
	if err != nil {
		return nil, err
	}
	var a Address
	copy(a[:], b)
	return &a, nil
}

// ReadUint64 will read a uint64 from the reader.
func ReadUint64(r io.Reader) (uint64, error) {
	b, err := ReadBytes(r, 8)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b), nil
}

// ReadBool will read a boolean from the reader.
// 00 if false else true.
func ReadBool(r io.Reader) (bool, error) {
	b, err := ReadBytes(r, 1)
	if err != nil {
		return false, err
	}
	return b[0] != byte(0x00), nil
}

// ReadBigInt reads a big int from the reader.
func ReadBigInt(r io.Reader) (*big.Int, error) {
	bigBytes, err := ReadByteArray(r)
	if err != nil {
		return nil, err
	}

	b := new(big.Int)
	b.SetBytes(bigBytes)
	return b, nil
}

// ReadHashArray reads an array of hashes from the reader.
func ReadHashArray(r io.Reader) ([]*chainhash.Hash, error) {
	l, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	out := make([]*chainhash.Hash, l)
	for i := uint64(0); i < l; l++ {
		out[i], err = ReadHash(r)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// WriteHashArray will serialize a hash array and return bytes.
func WriteHashArray(hs []*chainhash.Hash) []byte {
	out := []byte{}
	out = append(out, WriteVarInt(uint64(len(hs)))...)
	for _, h := range hs {
		out = append(out, h[:]...)
	}
	return out
}
