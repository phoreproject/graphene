package chain

import (
	"fmt"
	"sync"

	"github.com/phoreproject/graphene/p2p"
	"github.com/phoreproject/graphene/pb"
)

// ShardMux handles the various different blockchains associated with different shards.
type ShardMux struct {
	lock         *sync.RWMutex
	managers     map[uint64]*ShardManager
	beaconClient pb.BlockchainRPCClient

	hn *p2p.HostNode
}

// NewShardMux creates a new shard multiplexer.
func NewShardMux(beaconClient pb.BlockchainRPCClient, hn *p2p.HostNode) *ShardMux {
	return &ShardMux{
		managers:     make(map[uint64]*ShardManager),
		lock:         new(sync.RWMutex),
		beaconClient: beaconClient,
		hn:           hn,
	}
}

// IsManaging checks if the shard mux is managing a certain shard.
func (sm *ShardMux) IsManaging(shardID uint64) bool {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	_, found := sm.managers[shardID]
	return found
}

// StartManaging starts managing a certain shard.
func (sm *ShardMux) StartManaging(shardID uint64, init ShardChainInitializationParameters) (*ShardManager, error) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	shardManager, err := NewShardManager(shardID, init, sm.beaconClient, sm.hn)
	if err != nil {
		return nil, err
	}
	sm.managers[shardID] = shardManager
	return shardManager, nil
}

// StopManaging stops managing a certain shard.
func (sm *ShardMux) StopManaging(shardID uint64) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	delete(sm.managers, shardID)
}

// GetManager gets the manager for a certain shard ID
func (sm *ShardMux) GetManager(shardID uint64) (*ShardManager, error) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	manager, found := sm.managers[shardID]
	if !found {
		return nil, fmt.Errorf("not currently tracking shard %d", shardID)
	}
	return manager, nil
}
