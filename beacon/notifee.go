package beacon

import "github.com/phoreproject/synapse/primitives"

// BlockchainNotifee is a blockchain notifee.
type BlockchainNotifee interface {
	ConnectBlock(*primitives.Block)
	CrosslinkCreated(crosslink *primitives.Crosslink, shardID uint64)
}

// RegisterNotifee registers a notifee for blockchain
func (b *Blockchain) RegisterNotifee(n BlockchainNotifee) {
	b.notifeeLock.Lock()
	defer b.notifeeLock.Unlock()
	b.notifees = append(b.notifees, n)
}

// UnregisterNotifee unregisters a notifee for blockchain.
func (b *Blockchain) UnregisterNotifee(n BlockchainNotifee) {
	b.notifeeLock.Lock()
	defer b.notifeeLock.Unlock()
	for i, other := range b.notifees {
		if other == n {
			b.notifees = append(b.notifees[:i], b.notifees[i+1:]...)
		}
	}
}