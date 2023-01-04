package p2p

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	mdnsdiscovery "github.com/libp2p/go-libp2p/p2p/discovery"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	discoveryutils "github.com/libp2p/go-libp2p-discovery"
)

// MessageHandler is a handler for a specific message.
type MessageHandler func(id peer.ID, msg proto.Message) error

// ProtocolHandler handles all of the messages, discovery, and shut down for each protocol.
type ProtocolHandler struct {
	// ID is the protocol being handled.
	ID protocol.ID

	// MaximumPeers is the maximum number of peers to connect to with this protocol ID. After this number of peers is
	// reached, we'll signal that we'll drop this peer if needed.
	MaximumPeers int

	// MinimumPeers is the minimum number of peers to maintain for this protocol ID. After this number of peers is reached,
	// we won't connect to any more.
	MinimumPeers int

	// host is the host to connect to.
	host *HostNode

	// connManager is the connection manager
	connManager *ConnectionManager

	discovery discovery.Discovery

	messageHandlers     map[protoreflect.FullName]MessageHandler
	messageHandlersLock sync.RWMutex

	outgoingMessages     map[peer.ID]chan proto.Message
	outgoingMessagesLock sync.RWMutex

	ctx context.Context

	notifees    []ConnectionManagerNotifee
	notifeeLock sync.Mutex
}

// ConnectionManagerNotifee is a notifee for the connection manager.
type ConnectionManagerNotifee interface {
	PeerConnected(peer.ID, network.Direction)
	PeerDisconnected(peer.ID)
}

// newProtocolHandler constructs a new protocol handler for a specific protocol ID.
func newProtocolHandler(ctx context.Context, id protocol.ID, maxPeers int, minPeers int, host *HostNode, connManager *ConnectionManager, disc discovery.Discovery) *ProtocolHandler {
	ph := &ProtocolHandler{
		ID:               id,
		MaximumPeers:     maxPeers,
		MinimumPeers:     minPeers,
		discovery:        disc,
		host:             host,
		messageHandlers:  make(map[protoreflect.FullName]MessageHandler),
		outgoingMessages: make(map[peer.ID]chan proto.Message),
		connManager:      connManager,
		ctx:              ctx,
		notifees:         make([]ConnectionManagerNotifee, 0),
	}

	host.setStreamHandler(id, ph.handleStream)

	go ph.findPeers()

	go ph.advertise()

	return ph
}

// ShouldAcceptIncoming returns true if we should findPeers any more peers for this protocol. This doesn't mean the number
// of peers that support this protocol can't go above MaximumPeers since some other protocol may still want to connect.
func (p *ProtocolHandler) ShouldAcceptIncoming() bool {
	numPeers := p.host.CountPeers(p.ID)
	return numPeers < p.MaximumPeers
}

func (p *ProtocolHandler) shouldConnectOutgoing() bool {
	numPeers := p.host.CountPeers(p.ID)
	return numPeers < p.MinimumPeers
}

func (p *ProtocolHandler) advertise() {
	// advertise until we disconnect
	discoveryutils.Advertise(p.ctx, p.discovery, string(p.ID))
}

// RegisterHandler registers a handler for a protocol.
func (p *ProtocolHandler) RegisterHandler(messageName protoreflect.FullName, handler MessageHandler) error {
	p.messageHandlersLock.Lock()
	defer p.messageHandlersLock.Unlock()
	if _, found := p.messageHandlers[messageName]; found {
		return fmt.Errorf("handler for message name %s already exists", messageName)
	}

	p.messageHandlers[messageName] = handler
	return nil
}

const findPeerCycle = 300 * time.Second

// HandlePeerFound handles any discovered peers.
func (p *ProtocolHandler) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == p.host.GetHost().ID() {
		return
	}
	if len(pi.Addrs) > 0 && p.shouldConnectOutgoing() {
		logrus.WithField("forProto", p.ID)
		err := p.connManager.Connect(pi)
		if err != nil {
			logrus.WithField("forProto", p.ID).Error(err)
		}
	}
}

// findPeers looks for peers advertising our protocol ID and connects to them if needed.
func (p *ProtocolHandler) findPeers() {

	if p.connManager.Options.MDNS.Enabled {
		service, err := mdnsdiscovery.NewMdnsService(p.ctx, p.host.GetHost(), p.connManager.Options.MDNS.Interval, fmt.Sprintf("phore-%s-discovery._udp", p.ID))
		if err != nil {
			logrus.Warn(err)
		}

		service.RegisterNotifee(p)
	}

	for {
		findPeerCtx, cancel := context.WithTimeout(p.ctx, findPeerCycle)

		peers, err := p.discovery.FindPeers(findPeerCtx, string(p.ID), discovery.Limit(p.MaximumPeers))
		if err != nil {
			logrus.WithField("type", "error finding peers").Error(err)
			cancel()
			break
		}
		// now we want to go through each peer and try to connect to them

		// wait for either the parent to finish (meaning we should stop accepting new peers) or the cycle to end
		// meaning we should find more peers
	peerLoop:
		for {
			select {
			case pi, ok := <-peers:
				if !ok {
					time.Sleep(time.Second * 5)
					break peerLoop
				}
				p.HandlePeerFound(pi)
			case <-p.ctx.Done():
				cancel()
				return
			case <-findPeerCtx.Done():
				break peerLoop
			}
		}
	}
}

func (p *ProtocolHandler) receiveMessages(id peer.ID, r io.Reader) {
	err := processMessages(p.ctx, r, func(message proto.Message) error {
		name := proto.MessageName(message)
		logrus.WithFields(logrus.Fields{
			"peer":  id,
			"msg":   name,
			"proto": id,
		}).Debug("received message")
		p.messageHandlersLock.RLock()
		if handler, found := p.messageHandlers[name]; found {
			p.messageHandlersLock.RUnlock()
			err := handler(id, message)
			if err != nil {
				return err
			}
		} else {
			p.messageHandlersLock.RUnlock()
		}
		return nil
	})
	if err != nil {
		p.notifeeLock.Lock()
		for _, n := range p.notifees {
			n.PeerDisconnected(id)
		}
		p.notifeeLock.Unlock()
		if !strings.Contains(err.Error(), "stream reset") {
			logrus.WithField("type", "receiving messages from peer").WithField("peer", id).Error(err)
		}
	}
}

func (p *ProtocolHandler) sendMessages(id peer.ID, w io.Writer) {
	msgChan := make(chan proto.Message)

	p.outgoingMessagesLock.Lock()
	p.outgoingMessages[id] = msgChan
	p.outgoingMessagesLock.Unlock()

	go func() {
		for msg := range msgChan {
			err := writeMessage(msg, w)
			if err != nil {
				logrus.WithField("type", "sending message").WithField("toPeer", id).Error(err)

				p.notifeeLock.Lock()
				for _, n := range p.notifees {
					n.PeerDisconnected(id)
				}
				p.notifeeLock.Unlock()

				_ = p.host.DisconnectPeer(id)
			}
		}
	}()
}

func (p *ProtocolHandler) handleStream(s network.Stream) {
	p.sendMessages(s.Conn().RemotePeer(), s)

	logrus.WithField("from", s.Conn().RemotePeer()).Info("handling messages")

	go p.receiveMessages(s.Conn().RemotePeer(), s)

	p.notifeeLock.Lock()
	for _, n := range p.notifees {
		n.PeerConnected(s.Conn().RemotePeer(), s.Stat().Direction)
	}
	p.notifeeLock.Unlock()
}

// SendMessage writes a message to a peer.
func (p *ProtocolHandler) SendMessage(toPeer peer.ID, msg proto.Message) error {
	logrus.WithFields(logrus.Fields{
		"peer":  toPeer,
		"msg":   proto.MessageName(msg),
		"proto": p.ID,
	}).Debug("sending message")
	p.outgoingMessagesLock.RLock()
	msgsChan, found := p.outgoingMessages[toPeer]
	p.outgoingMessagesLock.RUnlock()
	if !found {
		return fmt.Errorf("not tracking peer %s", toPeer)
	}

	msgsChan <- msg
	return nil
}

// Listen is called when we start listening on an address.
func (p *ProtocolHandler) Listen(network.Network, multiaddr.Multiaddr) {}

// ListenClose is called when we stop listening on an address.
func (p *ProtocolHandler) ListenClose(network.Network, multiaddr.Multiaddr) {}

// Connected is called when we connect to a peer.
func (p *ProtocolHandler) Connected(net network.Network, conn network.Conn) {
	if conn.Stat().Direction != network.DirOutbound {
		return
	}

	_ = p.host.OpenStreams(conn.RemotePeer(), p.ID)
}

// Disconnected is called when we disconnect to a peer.
func (p *ProtocolHandler) Disconnected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()

	if net.Connectedness(peerID) == network.NotConnected {
		p.outgoingMessagesLock.Lock()
		defer p.outgoingMessagesLock.Unlock()
		if handler, found := p.outgoingMessages[peerID]; found {
			close(handler)
			delete(p.outgoingMessages, peerID)
		}
	}
}

// OpenedStream is called when we open a stream to a peer.
func (p *ProtocolHandler) OpenedStream(network.Network, network.Stream) {}

// ClosedStream is called when we close a stream to a peer.
func (p *ProtocolHandler) ClosedStream(network.Network, network.Stream) {}

// Notify notifies a specific notifier when certain events happen.
func (p *ProtocolHandler) Notify(n ConnectionManagerNotifee) {
	p.notifeeLock.Lock()
	p.notifees = append(p.notifees, n)
	p.notifeeLock.Unlock()
}

// StopNotify stops notifying a certain notifee about certain events.
func (p *ProtocolHandler) StopNotify(n ConnectionManagerNotifee) {
	p.notifeeLock.Lock()
	found := -1
	for i, notif := range p.notifees {
		if notif == n {
			found = i
			break
		}
	}
	if found != -1 {
		p.notifees = append(p.notifees[:found], p.notifees[found+1:]...)
	}
	p.notifeeLock.Unlock()
}
