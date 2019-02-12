package beacon

import (
	"fmt"

	"github.com/phoreproject/synapse/primitives"
)

// HandleNewBlocks handles any incoming blocks.
func (b *Blockchain) HandleNewBlocks(blocks chan primitives.Block) error {
	for {
		block := <-blocks
		err := b.ProcessBlock(&block)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
}
