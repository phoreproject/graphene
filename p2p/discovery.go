package p2p

import (
	"context"
	"github.com/sirupsen/logrus"
	"time"

	ps "github.com/libp2p/go-libp2p-peerstore"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
	p2pdiscovery "github.com/libp2p/go-libp2p-discovery"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
)

// MDNSOptions are options for the MDNS discovery mechanism.
type MDNSOptions struct {
	Enabled  bool
	Interval time.Duration
}

// DiscoveryOptions is the options used to discover peers
type DiscoveryOptions struct {
	// Optional. Each element is a peer address to connect with.
	PeerAddresses []ps.PeerInfo

	MDNS MDNSOptions
}

var activeDiscoveryNS = "synapse"

// NewDiscoveryOptions creates a DiscoveryOptions with default values
func NewDiscoveryOptions() DiscoveryOptions {
	return DiscoveryOptions{
		MDNS: MDNSOptions{
			Enabled:  true,
			Interval: 1 * time.Minute,
		},
	}
}

// Discovery is the service to discover other peers.
type Discovery struct {
	host    *HostNode
	options DiscoveryOptions
	ctx     context.Context
	p2pDiscovery p2pdiscovery.Discovery
}

// NewDiscovery creates a new discovery service.
func NewDiscovery(ctx context.Context, host *HostNode, options DiscoveryOptions) *Discovery {
	routing, err := kaddht.New(ctx, host.GetHost())
	if err != nil {
		panic(err)
	}
	return &Discovery{
		host:    host,
		ctx:     ctx,
		options: options,
		p2pDiscovery: p2pdiscovery.NewRoutingDiscovery(routing),
	}
}

// mDNSTag is the name of the mDNS service.
const mDNSTag = "_phore-discovery._udp"

// StartDiscovery protocols. Currently, this supports discovery via multicast
// DNS peer discovery.
//
// TODO: add other discovery protocols such as DHT, etc.
func (d Discovery) StartDiscovery() error {
	if d.options.MDNS.Enabled {
		err := d.discoverFromMDNS()
		if err != nil {
			logrus.Error(err)
		}
	}

	d.startActiveDiscovery()

	for _, p := range d.options.PeerAddresses {
		d.HandlePeerFound(p)
	}

	return nil
}

func (d Discovery) discoverFromMDNS() error {
	mdnsService, err := mdns.NewMdnsService(d.ctx, d.host.GetHost(), d.options.MDNS.Interval, mDNSTag)
	if err != nil {
		return err
	}

	mdnsService.RegisterNotifee(d)

	return nil
}

func (d Discovery) startActiveDiscovery() {
	p2pdiscovery.Advertise(d.ctx, d.p2pDiscovery, activeDiscoveryNS)
	peerInfo, err := d.p2pDiscovery.FindPeers(d.ctx, activeDiscoveryNS)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case pi := <-peerInfo:
				d.HandlePeerFound(pi)

			case <-d.ctx.Done():
				return
			}
		}
	}()
}

// HandlePeerFound registers the peer with the host.
func (d Discovery) HandlePeerFound(pi ps.PeerInfo) {
	d.host.PeerDiscovered(pi)
}
