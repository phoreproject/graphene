package p2p

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/multiformats/go-multiaddr"

	peer "github.com/libp2p/go-libp2p-peer"
	ps "github.com/libp2p/go-libp2p-peerstore"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
	logger "github.com/sirupsen/logrus"
)

// DiscoveryOptions is the options used to discover peers
type DiscoveryOptions struct {
	// Optional. An address file contains a list of peer addresses to connect with.
	// Each line has format: ID addr1 addr2...
	// Addresses from addr2 are optional. Each parts are separated by blank space
	// The address format for IPv4: /ip4/202.0.5.1/tcp/8080
	AddressFileNames []string
	// Optional. Each element is a peer address to connect with.
	PeerAddresses []ps.PeerInfo
	// Optional. A seed address is the URL to download a seed file from.
	// A seed file has the same format as AddressFileNames
	SeedAddresses []string

	UseMDNS bool
}

// NewDiscoveryOptions creates a DiscoveryOptions with default values
func NewDiscoveryOptions() *DiscoveryOptions {
	return &DiscoveryOptions{
		UseMDNS: true,
	}
}

func lineToPeerInfo(line string) (*ps.PeerInfo, error) {
	parts := strings.Split(line, " ")
	var id string
	var addresses []string

	for i, s := range parts {
		if i == 0 {
			id = strings.TrimSpace(s)
		} else {
			addresses = append(addresses, strings.TrimSpace(s))
		}
	}

	if id == "" {
		return nil, fmt.Errorf("Not found ID")
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("Not found address")
	}

	var peerInfo ps.PeerInfo
	peerInfo.ID = peer.ID(id)
	for _, a := range addresses {
		addr, err := multiaddr.NewMultiaddr(a)
		if err == nil {
			peerInfo.Addrs = append(peerInfo.Addrs, addr)
		}
	}
	return &peerInfo, nil
}

// Discovery interval for multicast DNS querying.
var discoveryInterval = 1 * time.Minute

// mDNSTag is the name of the mDNS service.
var mDNSTag = mdns.ServiceTag

// StartDiscovery protocols. Currently, this supports discovery via multicast
// DNS peer discovery.
//
// TODO(287): add other discovery protocols such as DHT, etc.
func startDiscovery(node *HostNode, options *DiscoveryOptions) error {
	if len(options.PeerAddresses) > 0 {
		go discoverFromPeerInfos(node, options.PeerAddresses)
	}

	if len(options.AddressFileNames) > 0 {
		go discoverFromFiles(node, options.AddressFileNames)
	}

	if options.UseMDNS {
		err := discoverFromMDNS(node)
		if err != nil {
			return err
		}
	}

	return nil
}

func discoverFromFiles(node *HostNode, fileNames []string) {
	for _, fileName := range fileNames {
		file, err := os.Open(fileName)
		if err != nil {
			logger.Error(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			peerInfo, err := lineToPeerInfo(line)
			if err != nil {
				logger.Error(err)
			} else {
				node.Connect(peerInfo)
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Error(err)
		}
	}
}

func discoverFromPeerInfos(node *HostNode, peerInfoList []ps.PeerInfo) {
}

func discoverFromLines(node *HostNode, lines []string) {
	for _, a := range lines {
		peerInfo, err := lineToPeerInfo(a)
		if err == nil {
			node.Connect(peerInfo)
		}
	}
}

func discoverFromMDNS(node *HostNode) error {
	mdnsService, err := mdns.NewMdnsService(node.GetContext(), node.GetHost(), discoveryInterval, mDNSTag)
	if err != nil {
		return err
	}

	mdnsService.RegisterNotifee(&discovery{node})

	return nil
}

// Discovery implements mDNS notifee interface.
type discovery struct {
	node *HostNode
}

// HandlePeerFound registers the peer with the host.
func (d *discovery) HandlePeerFound(pi ps.PeerInfo) {
	d.node.Connect(&pi)
}
