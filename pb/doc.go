//go:generate protoc -I . beaconrpc.proto --go_out=plugins=grpc:.
//go:generate protoc -I . shardrpc.proto --go_out=plugins=grpc:.
//go:generate protoc -I . p2p.proto --go_out=plugins=grpc:.
//go:generate protoc -I . common.proto --go_out=plugins=grpc:.
//go:generate protoc -I . relayerrpc.proto --go_out=plugins=grpc:.

package pb
