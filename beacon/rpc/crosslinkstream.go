package rpc

import (
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/primitives"
)

// CrosslinkStream tracks new crosslinks created.
type CrosslinkStream struct {
	trackingShard uint64
	callback func(c *primitives.Crosslink)
}

// NewCrosslinkStream tracks a new shard with the given callback.
func NewCrosslinkStream(trackingShard uint64, cb func(c *primitives.Crosslink)) *CrosslinkStream {
	return &CrosslinkStream{
		trackingShard: trackingShard,
		callback: cb,
	}
}

// ConnectBlock implements beacon.BlockchainNotifee and does nothing.
func (CrosslinkStream) ConnectBlock(*primitives.Block) {}

// CrosslinkCreated implements beacon.BlockchainNotifee and sends a message on the listening channel if we care about
// the new crosslink.
func (c CrosslinkStream) CrosslinkCreated(crosslink *primitives.Crosslink, shard uint64) {
	if c.trackingShard != shard {
		return
	}

	c.callback(crosslink)
}

var _ beacon.BlockchainNotifee = &CrosslinkStream{}