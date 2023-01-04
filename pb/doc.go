//go:generate protoc -I . beaconrpc.proto --go-grpc_out=. --go_out=.
//go:generate protoc -I . shardrpc.proto --go-grpc_out=. --go_out=.
//go:generate protoc -I . p2p.proto --go_out=.
//go:generate protoc -I . common.proto --go_out=.
//go:generate protoc -I . relayerrpc.proto --go-grpc_out=. --go_out=.

package pb
