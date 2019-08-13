package chain

import "github.com/phoreproject/synapse/chainhash"

// ShardChainInitializationParameters are the initialization parameters from the crosslink.
type ShardChainInitializationParameters struct {
	RootBlockHash chainhash.Hash
	RootSlot      uint64
}

// ShardManager represents part of the blockchain on a specific shard (starting from a specific crosslinked block hash),
// and continued to a certain point.
type ShardManager struct {
	ShardID                  uint64
	Chain                    *ShardChain
	Index                    *ShardBlockIndex
	InitializationParameters ShardChainInitializationParameters
}

// NewShardManager initializes a new shard manager responsible for keeping track of a shard chain.
func NewShardManager(shardID uint64, init ShardChainInitializationParameters) *ShardManager {
	return &ShardManager{
		ShardID:                  shardID,
		Chain:                    NewShardChain(init.RootSlot),
		Index:                    NewShardBlockIndex(),
		InitializationParameters: init,
	}
}
