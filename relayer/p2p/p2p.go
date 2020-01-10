package p2p

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/phoreproject/synapse/p2p"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/relayer/mempool"
	"github.com/sirupsen/logrus"
)

// RelayerSyncManager keeps track of transactions and packages sync status with peers.
type RelayerSyncManager struct {
	hostNode    *p2p.HostNode
	shardNumber uint64

	protocol            *p2p.ProtocolHandler
	mempool             *mempool.ShardMempool
	packageSubscription *pubsub.Subscription
}

// NewRelayerSyncManager constructs a new relayer sync manager.
func NewRelayerSyncManager(hostNode *p2p.HostNode, mempool *mempool.ShardMempool, shardNumber uint64) (*RelayerSyncManager, error) {
	r := &RelayerSyncManager{
		hostNode:    hostNode,
		mempool:     mempool,
		shardNumber: shardNumber,
	}
	if err := r.registerP2P(); err != nil {
		return nil, err
	}
	return r, nil
}

// RelayerSyncProtocol is the relayer sync protocol for a specific version and shard.
func RelayerSyncProtocol(shardNumber uint64, version uint64) protocol.ID {
	return protocol.ID(fmt.Sprintf("/phore/relayer/%d/v%d", shardNumber, version))
}

func (r *RelayerSyncManager) onGetPackagesMessage(msg []byte, id peer.ID) {
	logrus.WithField("from", id).Debug("got get packages message")

	var getPackageMessage pb.GetPackagesMessage

	err := proto.Unmarshal(msg, &getPackageMessage)
	if err != nil {
		logrus.Warn(err)
		return
	}

	root, err := r.mempool.GetCurrentRoot()
	if err != nil {
		logrus.Warn(err)
		return
	}

	// ignore requests where the state root differs
	if !bytes.Equal(getPackageMessage.TipStateRoot, root[:]) {
		return
	}

	txPackage, err := r.mempool.GetTransactions(-1)
	if err != nil {
		logrus.Warn(err)
		return
	}

	err = r.protocol.SendMessage(id, &pb.PackageMessage{
		Package: txPackage.ToProto(),
	})
	if err != nil {
		logrus.Warn(err)
	}
}

func (r *RelayerSyncManager) onSubmitTransaction(id peer.ID, msg proto.Message) error {
	submitTxMessage := msg.(*pb.SubmitTransaction)

	return r.mempool.Add(submitTxMessage.Transaction.TransactionData)
}

func (r *RelayerSyncManager) registerP2P() error {
	handler, err := r.hostNode.RegisterProtocolHandler(RelayerSyncProtocol(r.shardNumber, 1), 16, 8)
	if err != nil {
		return err
	}

	r.protocol = handler

	sub, err := r.hostNode.SubscribeMessage(fmt.Sprintf("/packageRequests/%d", r.shardNumber), r.onGetPackagesMessage)
	if err != nil {
		return err
	}

	r.packageSubscription = sub

	err = r.protocol.RegisterHandler("pb.SubmitTransaction", r.onSubmitTransaction)
	if err != nil {
		return err
	}

	return nil
}
