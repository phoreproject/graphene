import logging

from phore.framework import tester, validatornode, shardnode
from phore.pb import common_pb2


class ShardSyncTest(tester.Tester):
    def __init__(self):
        logging.info(logging.INFO)

        super().__init__()

    def _do_run(self):
        beacon_nodes = [self.create_beacon_node() for _ in range(1)]

        beacon_nodes[0].start()

        beacon_nodes[0].wait_for_rpc()

        shard_node_configs = [shardnode.ShardConfig.from_beacon(beacon_nodes[0]) for _ in range(2)]
        shard_nodes = []
        for c in shard_node_configs:
            c.initial_shards = ['1']
            shard_nodes.append(self.create_shard_node(c))

        shard_nodes[0].start()
        shard_nodes[0].wait_for_rpc()

        shard_nodes[1].start()
        shard_nodes[1].wait_for_rpc()

        validator_node = self.create_validator_node(
            validatornode.ValidatorConfig.from_beacon_and_shard(beacon_nodes[0], shard_nodes[0], "0-255")
        )
        validator_node.start()
        validator_node.wait_for_rpc()

        shard_nodes[0].wait_for_slot(4, 1)

        shard_node_0_addr = shard_nodes[0].get_listening_addresses().Addresses[0]

        shard_nodes[1].connect(common_pb2.ConnectMessage(Address=shard_node_0_addr))

        shard_nodes[1].wait_for_slot(8, 1)


ex = ShardSyncTest()

ex.run()
