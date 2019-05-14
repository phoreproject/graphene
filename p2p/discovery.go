package p2p

import (
	"context"
	"time"

	ps "github.com/libp2p/go-libp2p-peerstore"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
	p2pdiscovery "github.com/libp2p/go-libp2p-discovery"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	logger "github.com/sirupsen/logrus"
	maddr "github.com/multiformats/go-multiaddr"
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
var defaultBootstrapAddrStrings = []string{
	"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
	"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
	"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
	"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
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
	p2pDiscovery p2pdiscovery.Discovery
}

// NewDiscovery creates a new discovery service.
func NewDiscovery(ctx context.Context, host *HostNode, options DiscoveryOptions) *Discovery {
	routing, err := kaddht.New(ctx, host.GetHost())
	if err != nil {
		panic(err)
	}
	var bootstrapConfig = kaddht.BootstrapConfig{
		Queries: 1,
		Period: time.Duration(5 * time.Minute),
		//Period: time.Duration(1 * time.Second),
		Timeout: time.Duration(10 * time.Second),
	}
	if err = routing.BootstrapWithConfig(ctx, bootstrapConfig); err != nil {
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
			logger.Error(err)
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
	d.bootstrapActiveDiscovery()
	d.startAdvertise()
	d.startFindPeers()
}

func (d Discovery) bootstrapActiveDiscovery() {
	for _, p := range defaultBootstrapAddrStrings {
		peerAddr, err := maddr.NewMultiaddr(p)
		if err != nil {
			logger.Errorf("bootstrapActiveDiscovery address error %s", err.Error())
		}
		peerinfo, _ := ps.InfoFromP2pAddr(peerAddr)

		if err = d.host.GetHost().Connect(d.ctx, *peerinfo); err != nil {
			logger.Errorf("bootstrapActiveDiscovery connect error %s", err.Error())
		} else {
			logger.Debugf("Connection established with bootstrap node: %v", *peerinfo)
		}
	}
}

func (d Discovery) startAdvertise() {
	go func() {
		for {
			ttl, err := d.p2pDiscovery.Advertise(d.ctx, activeDiscoveryNS)
			if err != nil {
				logger.Errorf("Error advertising %s: %s", activeDiscoveryNS, err.Error())
				if d.ctx.Err() != nil {
					return
				}

				select {
				case <-time.After(2 * time.Second):
					continue
				case <-d.ctx.Done():
					return
				}
			}

			wait := 7 * ttl / 8
wait = 1 // for test not to wait too long

			select {
			case <-time.After(wait):
				continue

			case <-d.ctx.Done():
				return
			}
		}
	}()
}

func (d Discovery) startFindPeers() {
	peerChan, err := d.p2pDiscovery.FindPeers(d.ctx, activeDiscoveryNS)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			for peer := range peerChan {
				if d.host.GetHost().ID() == peer.ID {
					continue
				}
				logger.Debugf("Found active peer: %s", peer.String())
				d.HandlePeerFound(peer)
			}
			select {
			case <-time.After(1 * time.Second):
				continue

			case <-d.ctx.Done():
				return
			}
		}
	}()

}

// HandlePeerFound registers the peer with the host.
func (d Discovery) HandlePeerFound(pi ps.PeerInfo) {
	if d.host.GetHost().ID() == pi.ID {
		return
	}
	d.host.PeerDiscovered(pi)
}
