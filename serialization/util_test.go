package serialization_test

import (
	"bytes"
	"testing"

	"github.com/phoreproject/synapse/serialization"
)

func TestAppendAll(t *testing.T) {
	x := serialization.AppendAll([]byte("test"), []byte("test2"), []byte("test3"))
	xExpected := []byte("testtest2test3")
	if !bytes.Equal(x, xExpected) {
		t.Fatal("append all not behaving properly")
	}
}
