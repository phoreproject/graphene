package p2p

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/protocol"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"

	"github.com/libp2p/go-libp2p-core/peer"
	routingdiscovery "github.com/libp2p/go-libp2p-discovery"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
)

// MDNSOptions are options for the MDNS discovery mechanism.
type MDNSOptions struct {
	Enabled  bool
	Interval time.Duration
}

// ConnectionManagerOptions is the options used to discover peers
type ConnectionManagerOptions struct {
	// Optional. Each element is a peer address to connect with.
	BootstrapAddresses []peer.AddrInfo

	MDNS MDNSOptions
}

var activeDiscoveryNS = "synapse"

// NewConnectionManagerOptions creates a ConnectionManagerOptions with default values
func NewConnectionManagerOptions() ConnectionManagerOptions {
	return ConnectionManagerOptions{
		MDNS: MDNSOptions{
			Enabled:  false,
			Interval: 1 * time.Minute,
		},
	}
}


// ConnectionManager is the service to discover other peers.
type ConnectionManager struct {
	host         *HostNode
	options      ConnectionManagerOptions
	ctx          context.Context
	p2pDiscovery *routingdiscovery.RoutingDiscovery

	protocolConfiguration map[protocol.ID]*ProtocolHandler
	protocolConfigurationLock sync.RWMutex

	lastConnect map[peer.ID]time.Time
	lastConnectLock sync.RWMutex
}

// DHTProtocolID is the protocol ID used for DHT for Phore.
const DHTProtocolID = "/phore/kad/1.0.0"

// NewConnectionManager creates a new discovery service.
func NewConnectionManager(ctx context.Context, host *HostNode, discoveryOptions ConnectionManagerOptions) (*ConnectionManager, error) {
	routing, err := kaddht.New(ctx, host.GetHost(), dhtopts.Protocols(
		protocol.ID(DHTProtocolID),
	))
	if err != nil {
		return nil, err
	}
	var bootstrapConfig = kaddht.BootstrapConfig{
		Queries: 5,
		Period:  time.Duration(5 * time.Minute),
		Timeout: time.Duration(10 * time.Second),
	}
	if err = routing.BootstrapWithConfig(ctx, bootstrapConfig); err != nil {
		return nil, err
	}
	return &ConnectionManager{
		host:         host,
		ctx:          ctx,
		options:      discoveryOptions,
		p2pDiscovery: routingdiscovery.NewRoutingDiscovery(routing),
		protocolConfiguration: make(map[protocol.ID]*ProtocolHandler),
		lastConnect: make(map[peer.ID]time.Time),
	}, nil
}

const connectionTimeout = 10 * time.Second
const connectionCooldown = 60 * time.Second

// Connect connects to a peer.
func (cm *ConnectionManager) Connect(pi peer.AddrInfo) error {
	cm.lastConnectLock.Lock()
	defer cm.lastConnectLock.Unlock()
	lastConnect, found := cm.lastConnect[pi.ID]
	if !found || time.Since(lastConnect) > connectionCooldown {
		cm.lastConnect[pi.ID] = time.Now()
		ctx, _ := context.WithTimeout(context.Background(), connectionTimeout)
		return cm.host.Connect(ctx, pi)
	}
	return nil
}

// RegisterProtocolHandler registers a protocol handler.
func (cm *ConnectionManager) RegisterProtocolHandler(id protocol.ID, maxPeers int, minPeers int) (*ProtocolHandler, error) {
	cm.protocolConfigurationLock.Lock()
	defer cm.protocolConfigurationLock.Unlock()
	if _, found := cm.protocolConfiguration[id]; found {
		return nil, fmt.Errorf("protocol handler for ID %s already exists", id)
	}

	handler := newProtocolHandler(cm.ctx, id, maxPeers, minPeers, cm.host, cm, cm.p2pDiscovery)

	cm.host.Notify(handler)

	cm.protocolConfiguration[id] = handler
	return cm.protocolConfiguration[id], nil
}

// GetProtocols gets protocols handled by this connection manager.
func (cm *ConnectionManager) GetProtocols() []protocol.ID {
	cm.protocolConfigurationLock.RLock()
	defer cm.protocolConfigurationLock.RUnlock()

	protos := make([]protocol.ID, 0, len(cm.protocolConfiguration))
	for p := range cm.protocolConfiguration {
		protos = append(protos, p)
	}
	return protos
}

func (cm *ConnectionManager) HandleOutgoing(id protocol.ID, s network.Stream) error {
	cm.protocolConfigurationLock.RLock()
	ph, found := cm.protocolConfiguration[id]
	cm.protocolConfigurationLock.RUnlock()

	if !found {
		return fmt.Errorf("not tracking protocol %s", id)
	}

	ph.handleStream(s)
	return nil
}

// Listen is called when we start listening on a multiaddr.
func (cm *ConnectionManager) Listen(network.Network, multiaddr.Multiaddr) {}

// ListenClose is called when we stop listening on a multiaddr.
func (cm *ConnectionManager) ListenClose(network.Network, multiaddr.Multiaddr) {}

// Connected is called when we connect to a peer.
func (cm *ConnectionManager) Connected(net network.Network, conn network.Conn) {
	if conn.Stat().Direction != network.DirOutbound {
		return
	}

	err := cm.host.OpenStreams(conn.RemotePeer(), cm.GetProtocols()...)
	if err != nil {
		logrus.Error(err)
		_ = cm.host.DisconnectPeer(conn.RemotePeer())
	}
}

// Disconnected is called when we disconnect from a peer.
func (cm *ConnectionManager) Disconnected(net network.Network, conn network.Conn) {}

// OpenedStream is called when we open a stream.
func (cm *ConnectionManager) OpenedStream(network.Network, network.Stream) {}

// ClosedStream is called when we close a stream.
func (cm *ConnectionManager) ClosedStream(network.Network, network.Stream) {}