package p2p

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"reflect"

	proto "github.com/golang/protobuf/proto"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ps "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	multiaddr "github.com/multiformats/go-multiaddr"
	pb "github.com/phoreproject/synapse/pb"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// P2pHostNode is the node for host
type P2pHostNode struct {
	publicKey  crypto.PubKey
	privateKey crypto.PrivKey
	host       host.Host
	gossipSub  *pubsub.PubSub
	ctx        context.Context
	cancel     context.CancelFunc
	grpcServer *grpc.Server
	peerList   []P2pPeerNode
}

var protocolID = protocol.ID("/grpc/0.0.1")

// NewHostNode creates a host node
func NewHostNode(
	listenAddress multiaddr.Multiaddr,
	publicKey crypto.PubKey,
	privateKey crypto.PrivKey,
	server pb.MainRPCServer,
) (*P2pHostNode, error) {
	ctx, cancel := context.WithCancel(context.Background())
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(listenAddress),
		libp2p.Identity(privateKey),
	)
	if err != nil {
		logger.WithField("Function", "NewHostNode").Warn(err)
		cancel()
		return nil, err
	}

	for _, a := range host.Addrs() {
		logger.WithField("address", a.String()).Debug("binding to port")
	}

	if err != nil {
		cancel()
		return nil, err
	}

	g, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		cancel()
		return nil, err
	}

	grpcServer := grpc.NewServer()
	hostNode := P2pHostNode{
		publicKey:  publicKey,
		privateKey: privateKey,
		host:       host,
		gossipSub:  g,
		ctx:        ctx,
		cancel:     cancel,
		grpcServer: grpcServer,
	}

	host.SetStreamHandler(protocolID, hostNode.handleStream)

	pb.RegisterMainRPCServer(grpcServer, server)

	return &hostNode, nil
}

func writeMessage(message proto.Message, writer *bufio.Writer) error {
	data, err := proto.Marshal(message)
	if err != nil {
		logger.WithField("Function", "writeMessage").Warn(err)
		return err
	}

	logger.WithField("Function", "writeMessage").Info("Sending")
	messageName := proto.MessageName(message)
	nameBytes := []byte(messageName)
	nameLength := len(nameBytes)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(len(data)+1+nameLength))
	written, err := writer.Write(buf)
	if err != nil {
		logger.WithField("Function", "writeMessage").Warn(err)
		return err
	}
	if written != len(buf) {
		logger.WithField("Function", "writeMessage").Warn("written < len(buf)")
		return nil
	}

	writer.WriteByte(byte(nameLength))

	written, err = writer.Write(nameBytes)
	if err != nil {
		logger.WithField("Function", "writeMessage").Warn(err)
		return err
	}
	if written != len(nameBytes) {
		logger.WithField("Function", "writeMessage").Warn("written < len(nameBytes)")
		return nil
	}

	written, err = writer.Write(data)
	if err != nil {
		logger.WithField("Function", "writeMessage").Warn(err)
		return err
	}
	if written != len(data) {
		logger.WithField("Function", "writeMessage").Warn("written < len(data)")
		return nil
	}

	writer.Flush()

	proto.Unmarshal(data, message)

	logger.WithField("Function", "writeMessage").Debugf("Written message %s %v Length: %d", messageName, reflect.TypeOf(message), len(data))

	return nil
}

func readMessage(length uint32, reader *bufio.Reader) (proto.Message, error) {
	nameLength, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	nameBytes := make([]byte, nameLength)
	reader.Read(nameBytes)
	messageName := string(nameBytes)
	messageType := proto.MessageType(messageName)
	message := reflect.New(messageType.Elem()).Interface().(proto.Message)
	if message == nil {
		logger.WithField("Function", "readMessage").Warnf("Message is nil! %s %v", messageName, messageType)
	}

	data := make([]byte, length-1-uint32(nameLength))
	reader.Read(data)
	//logger.WithField("Function", "readMessage").Debugf("Read message %s %v Length: %d %v", messageName, reflect.TypeOf(message), len(data), reflect.New(messageType.Elem()).Interface())
	proto.Unmarshal(data, message)

	return message, nil
}

func readData(stream inet.Stream) {
	const stateReadHeader = 1
	const stateReadMessage = 2

	streamReader := NewStreamReader(stream)
	headerBuffer := make([]byte, 4)
	state := stateReadHeader
	var messageBuffer []byte
	for {
		switch state {
		case stateReadHeader:
			if streamReader.Read(headerBuffer) {
				fmt.Println("Received message header")
				messageLength := binary.LittleEndian.Uint32(headerBuffer)
				messageBuffer = make([]byte, messageLength)
				state = stateReadMessage
			}
			break

		case stateReadMessage:
			if streamReader.Read(messageBuffer) {
				state = stateReadHeader
				message, _ := readMessage(uint32(len(messageBuffer)), bufio.NewReader(bytes.NewReader(messageBuffer)))
				fmt.Println("Received message: " + proto.MessageName(message))
			}
			break
		}
	}
}

// handleStream handles an incoming stream.
func (node *P2pHostNode) handleStream(stream inet.Stream) {
	go readData(stream)
}

// Connect connects to a peer
func (node *P2pHostNode) Connect(peerInfo *peerstore.PeerInfo) (*P2pPeerNode, error) {
	for _, p := range node.GetHost().Peerstore().PeersWithAddrs() {
		if p == peerInfo.ID {
			logger.WithField("Function", "Connect").Warn("Already connected")
			return nil, nil
		}
	}

	logger.WithField("addrs", peerInfo.Addrs).WithField("id", peerInfo.ID).Debug("attempting to connect to a peer")

	node.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, ps.PermanentAddrTTL)

	stream, err := node.host.NewStream(context.Background(), peerInfo.ID, protocolID)

	if err != nil {
		logger.WithField("Function", "Connect").WithField("error", err).Warn("failed to open stream")
		return nil, err
	}

	go readData(stream)

	peerNode := NewP2pPeerNode(stream)

	node.peerList = append(node.peerList, peerNode)

	return &peerNode, err
}

// Run runs the main loop of the host node
func (node *P2pHostNode) Run() {
}

// GetGRPCServer returns the grpc server.
func (node *P2pHostNode) GetGRPCServer() *grpc.Server {
	return node.grpcServer
}

// GetPublicKey returns the public key
func (node *P2pHostNode) GetPublicKey() *crypto.PubKey {
	return &node.publicKey
}

// GetContext returns the context
func (node *P2pHostNode) GetContext() context.Context {
	return node.ctx
}

// GetHost returns the host
func (node *P2pHostNode) GetHost() host.Host {
	return node.host
}

// GetPeerList returns the peer list
func (node *P2pHostNode) GetPeerList() []P2pPeerNode {
	return node.peerList
}

// GetConnectedPeerCount returns the connected peer count
func (node *P2pHostNode) GetConnectedPeerCount() int {
	return node.host.Peerstore().Peers().Len()
}

// Broadcast broadcasts a message to the network for a topic.
func (node *P2pHostNode) Broadcast(topic string, data []byte) error {
	return node.gossipSub.Publish(topic, data)
}

// SubscribeMessage registers a handler for a network topic.
func (node *P2pHostNode) SubscribeMessage(topic string, handler func([]byte) error) (*pubsub.Subscription, error) {
	subscription, err := node.gossipSub.Subscribe(topic)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			msg, err := subscription.Next(node.ctx)
			if err != nil {
				logger.WithField("error", err).Warn("error when getting next topic message")
				return
			}

			err = handler(msg.Data)
			if err != nil {
				logger.WithField("topic", topic).WithField("error", err).Warn("error when handling message")
			}
		}
	}()

	return subscription, nil
}

// UnsubscribeMessage cancels a subscription to a topic.
func (node *P2pHostNode) UnsubscribeMessage(subscription *pubsub.Subscription) {
	subscription.Cancel()
}
