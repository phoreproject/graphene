package primitives_test

import (
	"bytes"
	"testing"

	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/primitives"
)

func TestForkDataSSZ(t *testing.T) {
	data := primitives.ForkData{
		PreForkVersion:  5,
		PostForkVersion: 100,
		ForkSlotNumber:  17,
	}
	buf := bytes.Buffer{}
	ssz.Encode(&buf, data)

	newData := primitives.ForkData{}
	ssz.Decode(bytes.NewReader(buf.Bytes()), &newData)
	if newData.PreForkVersion != data.PreForkVersion {
		t.Fatalf("ForkData: newData.PreForkVersion %d != data.PreForkVersion %d", newData.PreForkVersion, data.PreForkVersion)
	}
	if newData.PostForkVersion != data.PostForkVersion {
		t.Fatalf("ForkData: newData.PostForkVersion %d != data.PostForkVersion %d", newData.PostForkVersion, data.PostForkVersion)
	}
	if newData.ForkSlotNumber != data.ForkSlotNumber {
		t.Fatalf("ForkData: newData.ForkSlotNumber %d != data.ForkSlotNumber %d", newData.ForkSlotNumber, data.ForkSlotNumber)
	}
}
