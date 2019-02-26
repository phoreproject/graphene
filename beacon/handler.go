package beacon

import (
	"fmt"

	"github.com/phoreproject/prysm/shared/ssz"

	"github.com/phoreproject/synapse/primitives"
)

// HandleNewBlocks handles any incoming blocks.
func (b *Blockchain) HandleNewBlocks(blocks chan primitives.Block) error {
	for {
		block := <-blocks

		blockHash, err := ssz.TreeHash(block)
		if err != nil {
			continue
		}
		if blockHash == b.Tip() {
			continue
		}

		err = b.ProcessBlock(&block)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
}
