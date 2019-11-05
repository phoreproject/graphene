package p2p

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/relayer/mempool"
)

// RelayerSyncManager keeps track of transactions and packages sync status with peers.
type RelayerSyncManager struct {
	hostNode *p2p.HostNode
	shardNumber uint64

	protocol *p2p.ProtocolHandler
	mempool *mempool.ShardMempool
}

// NewRelayerSyncManager constructs a new relayer sync manager.
func NewRelayerSyncManager(hostNode *p2p.HostNode, mempool *mempool.ShardMempool, shardNumber uint64) (*RelayerSyncManager, error) {
	r := &RelayerSyncManager{
		hostNode: hostNode,
		mempool: mempool,
		shardNumber: shardNumber,
	}
	if err := r.registerP2P(); err != nil {
		return nil, err
	}
	return r, nil
}

func relayerSyncProtocol(shardNumber uint64, version uint64) protocol.ID {
	return protocol.ID(fmt.Sprintf("/phore/relayer/%d/v%d", shardNumber, version))
}

func (r *RelayerSyncManager) onGetPackagesMessage(id peer.ID, msg proto.Message) error {
	getPackageMessage := msg.(*pb.GetPackagesMessage)

	root, err := r.mempool.GetCurrentRoot()
	if err != nil {
		return err
	}

	// ignore requests where the state root differs
	if !bytes.Equal(getPackageMessage.TipStateRoot, root[:]) {
		return nil
	}

	txPackage, err := r.mempool.GetTransactions(-1)
	if err != nil {
		return err
	}

	return r.protocol.SendMessage(id, &pb.PackageMessage{
		Package:              txPackage.ToProto(),
	})
}


func (r *RelayerSyncManager) registerP2P() error {
	handler, err := r.hostNode.RegisterProtocolHandler(relayerSyncProtocol(r.shardNumber, 1), 16, 8)
	if err != nil {
		return err
	}

	r.protocol = handler

	err = r.protocol.RegisterHandler("pb.GetPackagesMessage", r.onGetPackagesMessage)
	if err != nil {
		return err
	}

	return nil
}