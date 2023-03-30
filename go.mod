module github.com/phoreproject/synapse

go 1.18

require (
	github.com/beevik/ntp v0.2.0
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/c-bata/go-prompt v0.2.3
	github.com/decred/dcrd/dcrec/secp256k1 v1.0.2
	github.com/dgraph-io/badger v1.6.0
	github.com/fatih/color v1.7.0
	github.com/getsentry/sentry-go v0.7.0
	github.com/go-test/deep v1.0.4
	github.com/gogo/protobuf v1.3.0
	github.com/jinzhu/gorm v1.9.10
	github.com/labstack/echo v3.3.10+incompatible
	github.com/libp2p/go-libp2p v0.3.1
	github.com/libp2p/go-libp2p-core v0.2.2
	github.com/libp2p/go-libp2p-discovery v0.1.0
	github.com/libp2p/go-libp2p-kad-dht v0.2.1
	github.com/libp2p/go-libp2p-peerstore v0.1.3
	github.com/libp2p/go-libp2p-pubsub v0.1.1
	github.com/mattn/go-sqlite3 v1.11.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.0.4
	github.com/multiformats/go-multiaddr-net v0.0.1
	github.com/phoreproject/bls v0.0.0-20190821133044-da95d4798b09
	github.com/pkg/errors v0.9.1
	github.com/prysmaticlabs/go-ssz v0.0.0-20190917152816-977e011d625d
	github.com/sirupsen/logrus v1.4.2
	github.com/tetratelabs/wazero v1.0.1
	github.com/tevjef/go-runtime-metrics v0.0.0-20170326170900-527a54029307
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f
	google.golang.org/grpc v1.51.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/yaml.v2 v2.2.4
)

require (
	github.com/AndreasBriese/bbloom v0.0.0-20190823232136-616930265c33 // indirect
	github.com/btcsuite/btcd v0.0.0-20190629003639-c26ffa870fd8 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/dgryski/go-farm v0.0.0-20190423205320-6a90982ecee2 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/huin/goupnp v1.0.0 // indirect
	github.com/influxdata/influxdb v1.8.1 // indirect
	github.com/ipfs/go-cid v0.0.3 // indirect
	github.com/ipfs/go-datastore v0.1.0 // indirect
	github.com/ipfs/go-ipfs-util v0.0.1 // indirect
	github.com/ipfs/go-log v0.0.1 // indirect
	github.com/ipfs/go-todocounter v0.0.1 // indirect
	github.com/jackpal/gateway v1.0.5 // indirect
	github.com/jackpal/go-nat-pmp v1.0.1 // indirect
	github.com/jbenet/go-temp-err-catcher v0.0.0-20150120210811-aac704a3f4f2 // indirect
	github.com/jbenet/goprocess v0.1.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/koron/go-ssdp v0.0.0-20180514024734-4a0ed625a78b // indirect
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/libp2p/go-addr-util v0.0.1 // indirect
	github.com/libp2p/go-buffer-pool v0.0.2 // indirect
	github.com/libp2p/go-conn-security-multistream v0.1.0 // indirect
	github.com/libp2p/go-eventbus v0.1.0 // indirect
	github.com/libp2p/go-flow-metrics v0.0.1 // indirect
	github.com/libp2p/go-libp2p-autonat v0.1.0 // indirect
	github.com/libp2p/go-libp2p-circuit v0.1.1 // indirect
	github.com/libp2p/go-libp2p-kbucket v0.2.1 // indirect
	github.com/libp2p/go-libp2p-loggables v0.1.0 // indirect
	github.com/libp2p/go-libp2p-mplex v0.2.1 // indirect
	github.com/libp2p/go-libp2p-nat v0.0.4 // indirect
	github.com/libp2p/go-libp2p-record v0.1.1 // indirect
	github.com/libp2p/go-libp2p-routing v0.1.0 // indirect
	github.com/libp2p/go-libp2p-secio v0.2.0 // indirect
	github.com/libp2p/go-libp2p-swarm v0.2.1 // indirect
	github.com/libp2p/go-libp2p-transport-upgrader v0.1.1 // indirect
	github.com/libp2p/go-libp2p-yamux v0.2.1 // indirect
	github.com/libp2p/go-maddr-filter v0.0.5 // indirect
	github.com/libp2p/go-mplex v0.1.0 // indirect
	github.com/libp2p/go-msgio v0.0.4 // indirect
	github.com/libp2p/go-nat v0.0.3 // indirect
	github.com/libp2p/go-openssl v0.0.2 // indirect
	github.com/libp2p/go-reuseport v0.0.1 // indirect
	github.com/libp2p/go-reuseport-transport v0.0.2 // indirect
	github.com/libp2p/go-stream-muxer-multistream v0.2.0 // indirect
	github.com/libp2p/go-tcp-transport v0.1.0 // indirect
	github.com/libp2p/go-ws-transport v0.1.0 // indirect
	github.com/libp2p/go-yamux v1.2.3 // indirect
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mattn/go-isatty v0.0.9 // indirect
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/mattn/go-tty v0.0.0-20180907095812-13ff1204f104 // indirect
	github.com/miekg/dns v1.1.12 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v0.1.0 // indirect
	github.com/mr-tron/base58 v1.1.2 // indirect
	github.com/multiformats/go-base32 v0.0.3 // indirect
	github.com/multiformats/go-multiaddr-dns v0.0.3 // indirect
	github.com/multiformats/go-multiaddr-fmt v0.0.1 // indirect
	github.com/multiformats/go-multibase v0.0.1 // indirect
	github.com/multiformats/go-multihash v0.0.7 // indirect
	github.com/multiformats/go-multistream v0.1.0 // indirect
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/pkg/term v0.0.0-20190109203006-aa71e9d9e942 // indirect
	github.com/protolambda/zssz v0.1.3 // indirect
	github.com/prysmaticlabs/go-bitfield v0.0.0-20190825002834-fb724e897364 // indirect
	github.com/spacemonkeygo/spacelog v0.0.0-20180420211403-2296661a0572 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.0.1 // indirect
	github.com/whyrusleeping/base32 v0.0.0-20170828182744-c30ac30633cc // indirect
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1 // indirect
	github.com/whyrusleeping/go-logging v0.0.0-20170515211332-0457bb6b88fc // indirect
	github.com/whyrusleeping/go-notifier v0.0.0-20170827234753-097c5d47330f // indirect
	github.com/whyrusleeping/mafmt v1.2.8 // indirect
	github.com/whyrusleeping/mdns v0.0.0-20190823211037-23958d6311f0 // indirect
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7 // indirect
	github.com/whyrusleeping/timecache v0.0.0-20160911033111-cfcb2f1abfee // indirect
	go.opencensus.io v0.22.2 // indirect
	golang.org/x/text v0.4.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
)
