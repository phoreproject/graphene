package p2p

import (
	"context"
	"errors"
	"github.com/libp2p/go-libp2p"
	"sort"
	"sync"
	"time"

	protocol2 "github.com/libp2p/go-libp2p-core/protocol"

	"github.com/libp2p/go-libp2p-peerstore/pstoremem"

	"github.com/phoreproject/synapse/chainhash"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/go-phore-connmgr"
	"github.com/phoreproject/synapse/pb"
	logger "github.com/sirupsen/logrus"
)

// MessageHandler is a function to handle messages.
type MessageHandler func(peer *Peer, message proto.Message) error

type messageHandlerAndID struct {
	handler MessageHandler
	id      uint64
}

// Message is a single message from a single peer.
type Message struct {
	From    *Peer
	Message proto.Message
}

// HostNode is the node for p2p host
// It's the low level P2P communication layer, the App class handles high level protocols
// The RPC communication is hanlded by App, not HostNode
type HostNode struct {
	publicKey  crypto.PubKey
	privateKey crypto.PrivKey

	host      host.Host
	gossipSub *pubsub.PubSub
	ctx       context.Context
	cancel    context.CancelFunc

	timeoutInterval   time.Duration
	heartbeatInterval time.Duration

	// discovery handles peer discovery (mDNS, DHT, etc)
	discovery *Discovery

	// All peers that connected successfully with correct handshake
	peerList     []*Peer
	peerListLock *sync.Mutex

	peerIDList     []peer.ID
	peerIDListLock *sync.RWMutex

	// a messageHandler is called when a message with certain name is received
	messageHandlerMap map[string][]messageHandlerAndID
	handlerLock       *sync.RWMutex
	currentID         uint64

	maxPeers      int
	chainProvider ChainProvider
}

var protocolID = protocol.ID("/grpc/phore/0.0.1")

// NewHostNode creates a host node
func NewHostNode(listenAddress multiaddr.Multiaddr, publicKey crypto.PubKey, privateKey crypto.PrivKey, options DiscoveryOptions, timeoutInterval time.Duration, maxPeers int, heartbeatInterval time.Duration, chainProvider ChainProvider) (*HostNode, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ps := pstoremem.NewPeerstore()

	h, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(listenAddress),
		libp2p.Identity(privateKey),
		libp2p.EnableRelay(),
		libp2p.Peerstore(ps),
		libp2p.ConnectionManager(connmgr.NewConnManager(maxPeers, maxPeers, time.Second*5, ps, map[protocol2.ID]int{})),
	)

	if err != nil {
		cancel()
		return nil, err
	}

	addrs, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{
		ID: h.ID(),
		Addrs: []multiaddr.Multiaddr{
			listenAddress,
		},
	})
	if err != nil {
		cancel()
		return nil, err
	}

	for _, a := range addrs {
		logger.WithField("addr", a).Info("binding to address")
	}

	// setup gossipsub protocol
	g, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		cancel()
		return nil, err
	}

	hostNode := &HostNode{
		publicKey:         publicKey,
		privateKey:        privateKey,
		host:              h,
		gossipSub:         g,
		ctx:               ctx,
		cancel:            cancel,
		messageHandlerMap: make(map[string][]messageHandlerAndID),
		handlerLock:       new(sync.RWMutex),
		currentID:         0,
		timeoutInterval:   timeoutInterval,
		peerListLock:      new(sync.Mutex),
		heartbeatInterval: heartbeatInterval,
		maxPeers:          maxPeers,
		chainProvider:     chainProvider,
		peerIDList:        make([]peer.ID, 0),
		peerIDListLock:    new(sync.RWMutex),
	}

	discovery := NewDiscovery(ctx, hostNode, options)
	hostNode.discovery = discovery

	// setup phore protocol
	h.SetStreamHandler(protocolID, hostNode.handleStream)

	return hostNode, nil
}

// handleStream handles an incoming stream.
func (node *HostNode) handleStream(stream network.Stream) {
	_, err := node.setupPeerNode(stream, false)
	if err != nil {
		logger.Error("setup", err)
		return
	}
}

func (node *HostNode) handleMessage(peer *Peer, message proto.Message) error {
	node.handlerLock.RLock()
	handlerMap, found := node.messageHandlerMap[proto.MessageName(message)]
	node.handlerLock.RUnlock()
	if found {
		for _, handler := range handlerMap {
			err := handler.handler(peer, message)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Handler is a message handler representation.
type Handler struct {
	ID          uint64
	MessageName string
}

// RegisterMessageHandler registers a message handler
func (node *HostNode) RegisterMessageHandler(messageName string, handler MessageHandler) Handler {
	node.handlerLock.Lock()
	defer node.handlerLock.Unlock()
	_, ok := node.messageHandlerMap[messageName]
	if !ok {
		node.messageHandlerMap[messageName] = make([]messageHandlerAndID, 0)
	}

	node.messageHandlerMap[messageName] = append(node.messageHandlerMap[messageName], messageHandlerAndID{handler, node.currentID})

	node.currentID++

	return Handler{node.currentID - 1, messageName}
}

// RemoveMessageHandler deregisters a message handler.
func (node *HostNode) RemoveMessageHandler(handler Handler) {
	node.handlerLock.Lock()
	defer node.handlerLock.Unlock()
	oldHandlerMap := node.messageHandlerMap[handler.MessageName]
	newHandlerMap := make([]messageHandlerAndID, 0, len(oldHandlerMap)-1)

	for i := range oldHandlerMap {
		if oldHandlerMap[i].id != handler.ID {
			newHandlerMap = append(newHandlerMap, oldHandlerMap[i])
		}
	}

	node.messageHandlerMap[handler.MessageName] = newHandlerMap
}

// Connect connects to a peer that we're not already connected to.
func (node *HostNode) Connect(peerInfo peer.AddrInfo) (*Peer, error) {
	if peerInfo.ID == node.GetHost().ID() {
		return nil, errors.New("cannot connect to self")
	}

	node.peerIDListLock.RLock()
	if node.IsPeerConnected(peerInfo.ID) {
		return nil, nil
	}
	node.peerIDList = append(node.peerIDList, peerInfo.ID)
	node.peerIDListLock.RUnlock()

	if len(node.peerList) >= node.maxPeers {
		node.attemptToEvictConnection()
		if len(node.peerList) >= node.maxPeers {
			return nil, nil
		}
	}

	err := node.host.Connect(node.ctx, peerInfo)
	if err != nil {
		return nil, err
	}

	node.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)

	stream, err := node.host.NewStream(context.Background(), peerInfo.ID, protocolID)

	if err != nil {
		logger.WithField("Function", "Connect").WithField("error", err).Warn("failed to open stream")
		return nil, err
	}

	return node.setupPeerNode(stream, true)
}

type nodeEvictionCandidate struct {
	peer            *Peer
	lastMessageTime int64
	connectedTime   int64
	keyedNetGroup   uint64
}

type nodeSorter struct {
	candidates []nodeEvictionCandidate
	comparator func(nodeEvictionCandidate, nodeEvictionCandidate) bool
}

// Len is the length of candidate nodes
func (a nodeSorter) Len() int {
	return len(a.candidates)
}

// Less compares two candidate nodes
func (a nodeSorter) Less(i, j int) bool {
	return a.comparator(a.candidates[i], a.candidates[j])
}

// Swap swaps two candidate nodes.
func (a nodeSorter) Swap(i, j int) {
	a.candidates[i], a.candidates[j] = a.candidates[j], a.candidates[i]
}

func eraseLastKElements(candidates []nodeEvictionCandidate, k int, comparator func(nodeEvictionCandidate, nodeEvictionCandidate) bool) []nodeEvictionCandidate {
	if k >= len(candidates) {
		return []nodeEvictionCandidate{}
	}
	if k == 0 {
		return candidates
	}
	sorter := nodeSorter{
		candidates: candidates,
		comparator: comparator,
	}
	sort.Sort(sorter)

	return candidates[0:k]
}

func (node *HostNode) attemptToEvictConnection() {
	var candidates []nodeEvictionCandidate
	for _, p := range node.peerList {
		c := nodeEvictionCandidate{
			peer:            p,
			lastMessageTime: p.LastMessageTime.Unix(),
			connectedTime:   p.connectedTime,
			keyedNetGroup:   p.keyedNetGroup,
		}
		candidates = append(candidates, c)
	}

	// Deterministically select 2 peers to protect by netgroup.
	// An attacker cannot predict which netgroups will be protected
	// Note in Bitcoin it's 4 peers, we may change it to 4 as well.
	candidates = eraseLastKElements(
		candidates,
		2,
		func(a nodeEvictionCandidate, b nodeEvictionCandidate) bool {
			return a.keyedNetGroup < b.keyedNetGroup
		},
	)

	// Protect 2 nodes that most recently sent us messages.
	// Note in Bitcoin it's 4 peers, we may change it to 4 as well.
	candidates = eraseLastKElements(
		candidates,
		2,
		func(a nodeEvictionCandidate, b nodeEvictionCandidate) bool {
			return a.lastMessageTime < b.lastMessageTime
		},
	)

	// Protect the half of the remaining nodes which have been connected the longest.
	// This replicates the non-eviction implicit behavior, and precludes attacks that start later.
	candidates = eraseLastKElements(
		candidates,
		len(candidates)/2,
		func(a nodeEvictionCandidate, b nodeEvictionCandidate) bool {
			return a.connectedTime > b.connectedTime
		},
	)

	if len(candidates) == 0 {
		return
	}

	var maxGroup uint64
	var maxConnections int
	var maxConnectionTime int64

	mapNetGroupNodes := map[uint64][]nodeEvictionCandidate{}
	for _, c := range candidates {
		group, ok := mapNetGroupNodes[c.keyedNetGroup]
		if !ok {
			group = []nodeEvictionCandidate{}
		}
		group = append(group, c)
		mapNetGroupNodes[c.keyedNetGroup] = group

		groupTime := group[0].connectedTime
		if len(group) > maxConnections || len(group) == maxConnections && groupTime > maxConnectionTime {
			maxConnections = len(group)
			maxConnectionTime = groupTime
			maxGroup = c.keyedNetGroup
		}
	}

	group := mapNetGroupNodes[maxGroup]
	err := node.DisconnectPeer(group[0].peer)
	if err != nil {
		return
	}
}

// IsPeerConnected checks if a peer is connected
func (node *HostNode) IsPeerConnected(peerID peer.ID) bool {
	for _, p := range node.peerIDList {
		if p == peerID {
			return true
		}
	}
	return false
}

// ChainProvider is the interface from the blockchain to the host node packages.
type ChainProvider interface {
	Height() uint64
	GenesisHash() chainhash.Hash
}

// Run runs the main loop of the host node
func (node *HostNode) setupPeerNode(stream network.Stream, outbound bool) (*Peer, error) {
	peerNode := newPeer(outbound, stream.Conn().RemotePeer(), node, node.timeoutInterval, stream, node.heartbeatInterval)

	node.peerListLock.Lock()
	node.peerList = append(node.peerList, peerNode)
	node.peerListLock.Unlock()

	logger.WithField("peer", peerNode.ID.Pretty()).WithField("outbound", peerNode.Outbound).Info("connected to peer")

	peerIDBytes, err := node.host.ID().MarshalBinary()
	if err != nil {
		return nil, err
	}

	peerInfo := peer.AddrInfo{
		ID:    node.host.ID(),
		Addrs: node.host.Addrs(),
	}
	peerInfoBytes, err := peerInfo.MarshalJSON()
	if err != nil {
		return nil, err
	}

	genesisHash := node.chainProvider.GenesisHash()

	peerNode.SendMessage(&pb.VersionMessage{
		Version:     ClientVersion,
		PeerID:      peerIDBytes,
		PeerInfo:    peerInfoBytes,
		Height:      node.chainProvider.Height(),
		GenesisHash: genesisHash[:],
	})

	return peerNode, nil
}

// GetPublicKey returns the public key
func (node *HostNode) GetPublicKey() *crypto.PubKey {
	return &node.publicKey
}

// GetContext returns the context
func (node *HostNode) GetContext() context.Context {
	return node.ctx
}

// GetHost returns the host
func (node *HostNode) GetHost() host.Host {
	return node.host
}

// Broadcast broadcasts a message to the network for a topic.
func (node *HostNode) Broadcast(topic string, data []byte) error {
	return node.gossipSub.Publish(topic, data)
}

// SubscribeMessage registers a handler for a network topic.
func (node *HostNode) SubscribeMessage(topic string, handler func([]byte, peer.ID)) (*pubsub.Subscription, error) {
	subscription, err := node.gossipSub.Subscribe(topic)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			msg, err := subscription.Next(node.ctx)
			if err != nil {
				logger.WithField("error", err).Warn("error when getting next topic message")
				continue
			}

			handler(msg.Data, msg.GetFrom())
		}
	}()

	return subscription, nil
}

// UnsubscribeMessage cancels a subscription to a topic.
func (node *HostNode) UnsubscribeMessage(subscription *pubsub.Subscription) {
	subscription.Cancel()
}

func (node *HostNode) removePeer(peer *Peer) {
	node.peerListLock.Lock()
	for i, p := range node.peerList {
		if p == peer {
			node.peerList = append(node.peerList[:i], node.peerList[i+1:]...)
			break
		}
	}
	node.peerListLock.Unlock()

	node.peerIDListLock.Lock()
	for i, p := range node.peerIDList {
		if p == peer.ID {
			node.peerIDList = append(node.peerIDList[:i], node.peerIDList[i+1:]...)
			break
		}
	}
	node.peerIDListLock.Unlock()
	node.host.Peerstore().ClearAddrs(peer.ID)
}

// DisconnectPeer disconnects a peer
func (node *HostNode) DisconnectPeer(peer *Peer) error {
	peer.Disconnect()
	node.removePeer(peer)
	return nil
}

// FindPeerByID finds a peer node by ID, returns nil if not found
func (node *HostNode) FindPeerByID(id peer.ID) (*Peer, bool) {
	node.peerListLock.Lock()
	defer node.peerListLock.Unlock()
	for _, p := range node.peerList {
		if p.ID == id {
			return p, true
		}
	}
	return nil, false
}

// PeerDiscovered is run when peers are discovered.
func (node *HostNode) PeerDiscovered(pi peer.AddrInfo) {
	_, err := node.Connect(pi)
	if err != nil {
		logger.WithField("err", err).Debug("could not connect to peer")
	}
}

// Connected checks if the host node is connected.
func (node *HostNode) Connected() bool {
	node.peerListLock.Lock()
	defer node.peerListLock.Unlock()
	for _, p := range node.peerList {
		if !p.Connecting {
			return true
		}
	}
	return false
}

// PeersConnected checks how many peers are connected.
func (node *HostNode) PeersConnected() int {
	node.peerListLock.Lock()
	defer node.peerListLock.Unlock()
	peersConnected := 0
	for _, p := range node.peerList {
		if !p.Connecting {
			peersConnected++
		}
	}
	return peersConnected
}

// GetPeerList returns a list of all peers.
func (node *HostNode) GetPeerList() []*Peer {
	node.peerListLock.Lock()
	defer node.peerListLock.Unlock()
	return node.peerList
}

// GetPeerByID gets a peer by ID or returns nil if we aren't
// connected.
func (node *HostNode) GetPeerByID(id peer.ID) *Peer {
	node.peerListLock.Lock()
	defer node.peerListLock.Unlock()
	for _, p := range node.peerList {
		if id == p.ID {
			return p
		}
	}
	return nil
}

// StartDiscovery starts the host node discovering peers.
func (node *HostNode) StartDiscovery() error {
	return node.discovery.StartDiscovery()
}
