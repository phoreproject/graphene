package blockchain

import "github.com/phoreproject/synapse/primitives"

// HandleNewBlocks handles any incoming blocks.
func (b *Blockchain) HandleNewBlocks(blocks chan primitives.Block) {
	for {
		block := <-blocks
		b.ProcessBlock(&block)
	}
}
