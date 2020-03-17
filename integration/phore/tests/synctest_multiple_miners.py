from phore.framework import tester, validatornode, shardnode, beaconnode
from phore.pb import common_pb2

def connect_nodes(node1: beaconnode, node2: beaconnode):
    addr = node1.get_listening_addresses().Addresses[0]
    node2.connect(common_pb2.ConnectMessage(Address=addr))

class SyncTestChain(tester.Tester):
    """
    This package connects 4 nodes in a chain and ensures that if the block
    producer is on one end, the node on the other end stays in sync.
    """
    def __init__(self):
        super().__init__()

    def _do_run(self):
        beacon_nodes = [self.create_beacon_node() for _ in range(2)]

        beacon_nodes[0].start()
        beacon_nodes[1].start()

        beacon_nodes[0].wait_for_rpc()
        beacon_nodes[1].wait_for_rpc()

        connect_nodes(beacon_nodes[0], beacon_nodes[1])

        shard_node = self.create_shard_node(shardnode.ShardConfig.from_beacon(beacon_nodes[0]))
        shard_node.start()
        shard_node.wait_for_rpc()

        shard_node2 = self.create_shard_node(shardnode.ShardConfig.from_beacon(beacon_nodes[1]))
        shard_node2.start()
        shard_node2.wait_for_rpc()

        validator_node = self.create_validator_node(
            validatornode.ValidatorConfig.from_beacon_and_shard(beacon_nodes[0], shard_node, "0-127")
        )
        validator_node.start()
        validator_node.wait_for_rpc()

        validator_node2 = self.create_validator_node(
            validatornode.ValidatorConfig.from_beacon_and_shard(beacon_nodes[1], shard_node2, "128-255")
        )
        validator_node2.start()
        validator_node2.wait_for_rpc()

        beacon_nodes[0].wait_for_slot(16)
        beacon_nodes[1].wait_for_slot(16)

        self.reset()


ex = SyncTestChain()

ex.run()
