module github.com/phoreproject/synapse

require (
	github.com/btcsuite/btcd v0.0.0-20190209000034-12ce2fc7d321 // indirect
	github.com/btcsuite/btcutil v0.0.0-20190207003914-4c204d697803
	github.com/btcsuite/golangcrypto v0.0.0-20150304025918-53f62d9b43e8
	github.com/coreos/go-semver v0.2.0 // indirect
	github.com/fd/go-nat v1.0.0 // indirect
	github.com/go-test/deep v1.0.1
	github.com/golang/protobuf v1.2.0
	github.com/google/uuid v1.1.0 // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/ipfs/go-cid v0.9.0 // indirect
	github.com/ipfs/go-ipfs-addr v0.1.25
	github.com/ipfs/go-ipfs-util v1.2.8 // indirect
	github.com/jbenet/go-temp-err-catcher v0.0.0-20150120210811-aac704a3f4f2 // indirect
	github.com/jbenet/goprocess v0.0.0-20160826012719-b497e2f366b8 // indirect
	github.com/libp2p/go-addr-util v2.0.7+incompatible // indirect
	github.com/libp2p/go-buffer-pool v0.1.3 // indirect
	github.com/libp2p/go-conn-security v0.1.15 // indirect
	github.com/libp2p/go-conn-security-multistream v0.1.15 // indirect
	github.com/libp2p/go-flow-metrics v0.2.0 // indirect
	github.com/libp2p/go-libp2p v6.0.29+incompatible
	github.com/libp2p/go-libp2p-autonat v0.0.0-20190207233022-494f7fce997b // indirect
	github.com/libp2p/go-libp2p-circuit v2.3.2+incompatible // indirect
	github.com/libp2p/go-libp2p-crypto v2.0.5+incompatible
	github.com/libp2p/go-libp2p-discovery v0.0.0-20190207233013-a666b9cafd4c // indirect
	github.com/libp2p/go-libp2p-host v3.0.15+incompatible
	github.com/libp2p/go-libp2p-interface-connmgr v0.0.21 // indirect
	github.com/libp2p/go-libp2p-interface-pnet v3.0.0+incompatible // indirect
	github.com/libp2p/go-libp2p-loggables v1.1.24 // indirect
	github.com/libp2p/go-libp2p-metrics v2.1.7+incompatible // indirect
	github.com/libp2p/go-libp2p-nat v0.8.8 // indirect
	github.com/libp2p/go-libp2p-net v3.0.15+incompatible
	github.com/libp2p/go-libp2p-peer v2.4.0+incompatible
	github.com/libp2p/go-libp2p-peerstore v2.0.6+incompatible
	github.com/libp2p/go-libp2p-protocol v1.0.0
	github.com/libp2p/go-libp2p-pubsub v0.11.10
	github.com/libp2p/go-libp2p-routing v2.7.1+incompatible // indirect
	github.com/libp2p/go-libp2p-secio v2.0.17+incompatible // indirect
	github.com/libp2p/go-libp2p-swarm v3.0.22+incompatible // indirect
	github.com/libp2p/go-libp2p-transport v3.0.15+incompatible // indirect
	github.com/libp2p/go-libp2p-transport-upgrader v0.1.16 // indirect
	github.com/libp2p/go-maddr-filter v1.1.10 // indirect
	github.com/libp2p/go-mplex v0.2.30 // indirect
	github.com/libp2p/go-msgio v0.0.6 // indirect
	github.com/libp2p/go-reuseport-transport v0.2.0 // indirect
	github.com/libp2p/go-stream-muxer v3.0.1+incompatible // indirect
	github.com/libp2p/go-tcp-transport v2.0.16+incompatible // indirect
	github.com/libp2p/go-ws-transport v2.0.15+incompatible // indirect
	github.com/mattn/go-colorable v0.1.0 // indirect
	github.com/miekg/dns v1.1.4 // indirect
	github.com/minio/sha256-simd v0.0.0-20190131020904-2d45a736cd16 // indirect
	github.com/multiformats/go-multiaddr v1.4.0
	github.com/multiformats/go-multiaddr-net v1.7.1
	github.com/multiformats/go-multibase v0.3.0 // indirect
	github.com/multiformats/go-multistream v0.3.9 // indirect
	github.com/phoreproject/bls v0.0.0-20190219000203-afaefda3ea64
	github.com/phoreproject/prysm v0.0.0-20190208054636-b485eebbb134
	github.com/prysmaticlabs/prysm v0.0.0-20190208215336-7ae19ec37051 // indirect
	github.com/sirupsen/logrus v1.3.0
	github.com/stretchr/testify v1.3.0 // indirect
	github.com/whyrusleeping/base32 v0.0.0-20170828182744-c30ac30633cc // indirect
	github.com/whyrusleeping/go-notifier v0.0.0-20170827234753-097c5d47330f // indirect
	github.com/whyrusleeping/go-smux-multiplex v3.0.16+incompatible // indirect
	github.com/whyrusleeping/go-smux-multistream v2.0.2+incompatible // indirect
	github.com/whyrusleeping/go-smux-yamux v2.0.8+incompatible // indirect
	github.com/whyrusleeping/mafmt v1.2.8 // indirect
	github.com/whyrusleeping/mdns v0.0.0-20180901202407-ef14215e6b30 // indirect
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7 // indirect
	github.com/whyrusleeping/timecache v0.0.0-20160911033111-cfcb2f1abfee // indirect
	github.com/whyrusleeping/yamux v1.1.5 // indirect
	golang.org/x/arch v0.0.0-20181203225421-5a4828bb7045 // indirect
	golang.org/x/crypto v0.0.0-20190222235706-ffb98f73852f // indirect
	golang.org/x/net v0.0.0-20190206173232-65e2d4e15006
	golang.org/x/sys v0.0.0-20190222171317-cd391775e71e // indirect
	google.golang.org/grpc v1.18.0
)
