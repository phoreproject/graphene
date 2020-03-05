package beacon

import "github.com/phoreproject/synapse/primitives"

// BlockchainNotifee is a blockchain notifee.
type BlockchainNotifee interface {
	ConnectBlock(*primitives.Block)
	CrosslinkCreated(crosslink *primitives.Crosslink, shardID uint64)
}

// RegisterNotifee registers a notifee for blockchain
func (b *Blockchain) RegisterNotifee(n BlockchainNotifee) {
	b.Notifees = append(b.Notifees, n)
}

// UnregisterNotifee unregisters a notifee for blockchain.
func (b *Blockchain) UnregisterNotifee(n BlockchainNotifee) {
	for i, other := range b.Notifees {
		if other == n {
			b.Notifees = append(b.Notifees[:i], b.Notifees[i+1:]...)
		}
	}
}