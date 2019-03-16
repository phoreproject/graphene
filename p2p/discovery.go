package p2p

import (
	"context"
	"time"

	ps "github.com/libp2p/go-libp2p-peerstore"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
)

// MDNSOptions are options for the MDNS discovery mechanism.
type MDNSOptions struct {
	Enabled  bool
	Interval time.Duration
}

// DiscoveryOptions is the options used to discover peers
type DiscoveryOptions struct {
	// Optional. An address file contains a list of peer addresses to connect with.
	// Each line has format: ID addr1 addr2...
	// Addresses from addr2 are optional. Each parts are separated by blank space
	// The address format for IPv4: /ip4/202.0.5.1/tcp/8080
	AddressFileNames []string
	// Optional. Each element is a peer address to connect with.
	PeerAddresses []ps.PeerInfo
	// Optional. A seed address is the URL to download a seed file from.
	// A seed file has the same format as AddressFileNames
	SeedAddresses []string

	MDNS MDNSOptions
}

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
}

// NewDiscovery creates a new discovery service.
func NewDiscovery(ctx context.Context, host *HostNode, options DiscoveryOptions) *Discovery {
	return &Discovery{
		host:    host,
		ctx:     ctx,
		options: options,
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
			return err
		}
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

// HandlePeerFound registers the peer with the host.
func (d Discovery) HandlePeerFound(pi ps.PeerInfo) {
	d.host.PeerDiscovered(pi)
}
