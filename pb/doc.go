//go:generate protoc -I . rpc.proto --go_out=plugins=grpc:.
//go:generate protoc -I . p2p.proto --go_out=plugins=grpc:.

package pb
