// Code generated by protoc-gen-go. DO NOT EDIT.
// source: relayerrpc.proto

package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// RelayerRPCClient is the client API for RelayerRPC service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type RelayerRPCClient interface {
}

type relayerRPCClient struct {
	cc *grpc.ClientConn
}

func NewRelayerRPCClient(cc *grpc.ClientConn) RelayerRPCClient {
	return &relayerRPCClient{cc}
}

// RelayerRPCServer is the server API for RelayerRPC service.
type RelayerRPCServer interface {
}

func RegisterRelayerRPCServer(s *grpc.Server, srv RelayerRPCServer) {
	s.RegisterService(&_RelayerRPC_serviceDesc, srv)
}

var _RelayerRPC_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pb.RelayerRPC",
	HandlerType: (*RelayerRPCServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams:     []grpc.StreamDesc{},
	Metadata:    "relayerrpc.proto",
}

func init() { proto.RegisterFile("relayerrpc.proto", fileDescriptor_relayerrpc_a3453be37fb0461b) }

var fileDescriptor_relayerrpc_a3453be37fb0461b = []byte{
	// 61 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x28, 0x4a, 0xcd, 0x49,
	0xac, 0x4c, 0x2d, 0x2a, 0x2a, 0x48, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x2a, 0x48,
	0x32, 0xe2, 0xe1, 0xe2, 0x0a, 0x82, 0x88, 0x07, 0x05, 0x38, 0x27, 0xb1, 0x81, 0x25, 0x8c, 0x01,
	0x01, 0x00, 0x00, 0xff, 0xff, 0x69, 0x8b, 0x23, 0x9e, 0x2c, 0x00, 0x00, 0x00,
}