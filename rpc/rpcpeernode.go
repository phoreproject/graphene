package rpc

import (
	proto "github.com/golang/protobuf/proto"
)

// RpcPeerNode is the base interface of a peer node
type RpcPeerNode interface {
	Send(data []byte)
}

// SendMessage sends a protobuf message
func SendMessage(node RpcPeerNode, message proto.Message) {
	data, err := proto.Marshal(message)
	if err != nil {
	}
	node.Send(data)
}
