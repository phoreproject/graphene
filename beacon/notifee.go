package beacon

import "github.com/phoreproject/synapse/primitives"

// BlockchainNotifee is a blockchain notifee.
type BlockchainNotifee interface {
	ConnectBlock(*primitives.Block)
}

// RegisterNotifee registers a notifee for blockchain
func (b *Blockchain) RegisterNotifee(n BlockchainNotifee) {
	b.Notifees = append(b.Notifees, n)
}
