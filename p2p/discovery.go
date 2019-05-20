package p2p

import (
	"context"
	"time"

	protocol "github.com/libp2p/go-libp2p-protocol"

	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"

	p2pdiscovery "github.com/libp2p/go-libp2p-discovery"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	ps "github.com/libp2p/go-libp2p-peerstore"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/pb"
	logger "github.com/sirupsen/logrus"
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
	"/ip4/134.209.58.178/tcp/11781/ipfs/12D3KooWGeBbgdgVrfd6GhGscUp8LxAYGrvYSWpD4sHTNyrUYG3T",
	"/ip4/206.189.214.61/tcp/11781/ipfs/12D3KooWMFRdDoWiS7LS3J3pjgGtNYTDtTeanFFUCJyTKxiyk2KZ",
}

// NewDiscoveryOptions creates a DiscoveryOptions with default values
func NewDiscoveryOptions() DiscoveryOptions {
	return DiscoveryOptions{
		MDNS: MDNSOptions{
			Enabled:  false,
			Interval: 1 * time.Minute,
		},
	}
}

// Discovery is the service to discover other peers.
type Discovery struct {
	host         *HostNode
	options      DiscoveryOptions
	ctx          context.Context
	p2pDiscovery p2pdiscovery.Discovery
}

// DHTProtocolID is the protocol ID used for DHT for Phore.
const DHTProtocolID = "/phore/kad/1.0.0"

// NewDiscovery creates a new discovery service.
func NewDiscovery(ctx context.Context, host *HostNode, discoveryOptions DiscoveryOptions) *Discovery {
	routing, err := kaddht.New(ctx, host.GetHost(), dhtopts.Protocols(
		protocol.ID(DHTProtocolID),
	))
	if err != nil {
		panic(err)
	}
	var bootstrapConfig = kaddht.BootstrapConfig{
		Queries: 1,
		Period:  time.Duration(5 * time.Minute),
		//Period: time.Duration(1 * time.Second),
		Timeout: time.Duration(10 * time.Second),
	}
	if err = routing.BootstrapWithConfig(ctx, bootstrapConfig); err != nil {
		panic(err)
	}
	return &Discovery{
		host:         host,
		ctx:          ctx,
		options:      discoveryOptions,
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

	for _, addrString := range defaultBootstrapAddrStrings {
		addr, err := maddr.NewMultiaddr(addrString)
		if err != nil {
			logger.Error(err)
		}

		pinfo, err := ps.InfoFromP2pAddr(addr)
		if err != nil {
			logger.Error(err)
		}

		d.HandlePeerFound(*pinfo)
	}

	for _, pinfo := range d.options.PeerAddresses {
		d.HandlePeerFound(pinfo)
	}

	d.startActiveDiscovery()

	d.startGetAddr()

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
			logger.Infof("Connection established with bootstrap node: %v", *peerinfo)
		}
	}
}

func (d Discovery) startAdvertise() {
	go func() {
		for {
			ttl, err := d.p2pDiscovery.Advertise(d.ctx, activeDiscoveryNS)
			if err != nil {
				// it's error when there is no any peers yet, which is not an error.
				// so we print the log as debug instead of error to avoid spamming the log.
				logger.Debugf("Error advertising %s: %s", activeDiscoveryNS, err.Error())
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
			// Uncomment below line in testing to not to wait too long
			//wait = 1

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
				d.HandlePeerFound(peer)
			}
			select {
			case <-time.After(10 * time.Second):
				continue

			case <-d.ctx.Done():
				return
			}
		}
	}()

}

func (d Discovery) startGetAddr() {
	go func() {
		for {
			peerList := d.host.GetPeerList()
			for _, peer := range peerList {
				peer.SendMessage(&pb.GetAddrMessage{})
			}
			select {
			case <-time.After(60 * time.Second):
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
