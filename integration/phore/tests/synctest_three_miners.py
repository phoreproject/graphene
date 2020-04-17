from phore.framework import tester, validatornode, shardnode, beaconnode
from phore.pb import common_pb2
import time

def connect_nodes(node1: beaconnode, node2: beaconnode):
    addr = node1.get_listening_addresses().Addresses[0]
    node2.connect(common_pb2.ConnectMessage(Address=addr))

class SyncTestThreeMiners(tester.Tester):
    """
    This package connects 4 nodes in a chain and ensures that if the block
    producer is on one end, the node on the other end stays in sync.
    """
    def __init__(self):
        super().__init__()

    def _do_run(self):
        self.genesis_time = int(time.time() + 10)
        beacon_nodes = [self.create_beacon_node() for _ in range(3)]

        beacon_nodes[0].start()
        beacon_nodes[1].start()
        beacon_nodes[2].start()

        beacon_nodes[0].wait_for_rpc()
        beacon_nodes[1].wait_for_rpc()
        beacon_nodes[2].wait_for_rpc()

        connect_nodes(beacon_nodes[0], beacon_nodes[1])
        connect_nodes(beacon_nodes[1], beacon_nodes[2])
        connect_nodes(beacon_nodes[0], beacon_nodes[2])

        shard_nodes = [self.create_shard_node(shardnode.ShardConfig.from_beacon(beacon_nodes[i])) for i in range(3)]
        for s in shard_nodes:
          s.start()

        for s in shard_nodes:
          s.wait_for_rpc()

        validator_ranges = ["0-80", "81-140", "141-255"]

        validator_nodes = [self.create_validator_node(validatornode.ValidatorConfig.from_beacon_and_shard(beacon_nodes[i], shard_nodes[i], validator_ranges[i])) for i in range(3)]

        for v in validator_nodes:
          v.start()

        for v in validator_nodes:
          v.wait_for_rpc()

        beacon_nodes[0].wait_for_slot(32)
        beacon_nodes[1].wait_for_slot(32)

        self.reset()


ex = SyncTestThreeMiners()

ex.run()
