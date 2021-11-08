package csmt

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/phoreproject/graphene/chainhash"
)

func TestNodeSerializeDeserialize(t *testing.T) {
	nodes := []Node{
		{
			value:    chainhash.Hash{1, 2, 3},
			one:      true,
			oneKey:   &chainhash.Hash{1, 3, 5},
			oneValue: &chainhash.Hash{2, 4, 6},
			left:     nil,
			right:    nil,
		},
		{
			value:    chainhash.Hash{1, 2, 3},
			one:      false,
			oneKey:   nil,
			oneValue: nil,
			left:     &chainhash.Hash{2, 3, 4},
			right:    nil,
		},
		{
			value:    chainhash.Hash{1, 2, 3},
			one:      false,
			oneKey:   nil,
			oneValue: nil,
			left:     &chainhash.Hash{3, 4, 5},
			right:    &chainhash.Hash{4, 5, 6},
		},
		{
			value:    chainhash.Hash{1, 2, 3},
			one:      false,
			oneKey:   nil,
			oneValue: nil,
			left:     nil,
			right:    &chainhash.Hash{5, 6, 7},
		},
	}

	for _, node := range nodes {
		nodeSer := node.Serialize()

		nodeDeser, err := DeserializeNode(nodeSer)
		if err != nil {
			t.Fatal(err)
		}

		deep.Equal(node, *nodeDeser)
	}
}
