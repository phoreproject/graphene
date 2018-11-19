package serialization_test

import (
	"bytes"
	crand "crypto/rand"
	"math"
	"math/rand"
	"testing"

	"github.com/phoreproject/synapse/serialization"
)

func TestReadBytes(t *testing.T) {
	// test read bytes normal case
	r := bytes.NewBuffer([]byte("abc"))
	b, err := serialization.ReadBytes(r, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 3 {
		t.Fatal(err)
	}
	if !bytes.Equal(b, []byte("abc")) {
		t.Fatal("read bytes do not match bytes in buffer")
	}

	_, err = serialization.ReadBytes(r, 1)
	if err == nil {
		t.Fatal("read bytes longer than buffer")
	}
}

func TestReadWriteVarInt(t *testing.T) {
	r1 := uint64(rand.Uint32()) + math.MaxUint32

	r1out, err := serialization.ReadVarInt(bytes.NewBuffer(serialization.WriteVarInt(r1)))
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r1out {
		t.Fatalf("read value does not match written value (expected: %d, got: %d)", r1, r1out)
	}

	i := rand.Uint32()
	i >>= 16
	i16 := uint16(i)

	r2 := uint64(i16) + math.MaxUint16

	r2out, err := serialization.ReadVarInt(bytes.NewBuffer(serialization.WriteVarInt(r2)))
	if err != nil {
		t.Fatal(err)
	}
	if uint64(r2) != r2out {
		t.Fatalf("read value does not match written value (expected: %d, got: %d)", r2, r2out)
	}

	i = rand.Uint32()
	i >>= 24
	i8 := uint8(i)

	r3 := uint64(i8) + math.MaxUint8

	r3out, err := serialization.ReadVarInt(bytes.NewBuffer(serialization.WriteVarInt(r3)))
	if err != nil {
		t.Fatal(err)
	}
	if uint64(r3) != r3out {
		t.Fatalf("read value does not match written value (expected: %d, got: %d)", r3, r3out)
	}

	r4 := uint64(i >> 2) // make sure we're below 0xfd

	r4out, err := serialization.ReadVarInt(bytes.NewBuffer(serialization.WriteVarInt(r4)))
	if err != nil {
		t.Fatal(err)
	}
	if uint64(r4) != r4out {
		t.Fatalf("read value does not match written value (expected: %d, got: %d)", r4, r4out)
	}

	_, err = serialization.ReadVarInt(bytes.NewBuffer([]byte{}))
	if err == nil {
		t.Fatal("ReadVarInt succeeded in reading from empty buffer")
	}

	_, err = serialization.ReadVarInt(bytes.NewBuffer([]byte{0xfd, 0x00}))
	if err == nil {
		t.Fatal("ReadVarInt succeeded in reading from short buffer")
	}

	_, err = serialization.ReadVarInt(bytes.NewBuffer([]byte{0xfe, 0x00, 0x00, 0x00}))
	if err == nil {
		t.Fatal("ReadVarInt succeeded in reading from short buffer")
	}

	_, err = serialization.ReadVarInt(bytes.NewBuffer([]byte{0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}))
	if err == nil {
		t.Fatal("ReadVarInt succeeded in reading from short buffer")
	}
}

func TestReadWriteByteArray(t *testing.T) {
	b := make([]byte, 20)
	_, err := crand.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	b2, err := serialization.ReadByteArray(bytes.NewBuffer(serialization.WriteByteArray(b)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b2, b) {
		t.Fatal("bytes do not match")
	}

	bErr := bytes.NewBuffer([]byte{0xfd, 0x00})
	_, err = serialization.ReadByteArray(bErr)
	if err == nil {
		t.Fatal(err)
	}
}

func TestReadWriteUint32Array(t *testing.T) {
	b := make([]uint32, 30)
	for i := range b {
		b[i] = rand.Uint32()
	}

	bSer := bytes.NewBuffer(serialization.WriteUint32Array(b))
	b2, err := serialization.ReadUint32Array(bSer)
	if err != nil {
		t.Fatal(err)
	}
	if len(b2) != len(b) {
		t.Fatal("new uint32 array does not match (length)")
	}
	for i := range b2 {
		if b[i] != b2[i] {
			t.Fatal("new uint32 array does not match")
		}
	}

	bErr1 := bytes.NewBuffer([]byte{0xfd, 0x00})
	if _, err = serialization.ReadUint32Array(bErr1); err == nil {
		t.Fatal("invalid varint doesn't fail")
	}

	bErr2 := bytes.NewBuffer([]byte{0x03, 0x00})
	if _, err = serialization.ReadUint32Array(bErr2); err == nil {
		t.Fatal("invalid length doesn't error")
	}
}
