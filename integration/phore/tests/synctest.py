from phore.framework import tester, validatornode, shardnode
from phore.pb import common_pb2


class Example(tester.Tester):
    def __init__(self):
        super().__init__()

    def _do_run(self):
        beacon_nodes = [self.create_beacon_node() for _ in range(2)]

        beacon_nodes[0].start()
        beacon_nodes[1].start()

        beacon_nodes[0].wait_for_rpc()
        beacon_nodes[1].wait_for_rpc()

        addrs = beacon_nodes[0].get_listening_addresses().Addresses

        shard_node = self.create_shard_node(shardnode.ShardConfig.from_beacon(beacon_nodes[0]))
        shard_node.start()
        shard_node.wait_for_rpc()

        validator_node = self.create_validator_node(
            validatornode.ValidatorConfig.from_beacon_and_shard(beacon_nodes[0], shard_node, "0-255")
        )
        validator_node.start()
        validator_node.wait_for_rpc()

        beacon_nodes[0].wait_for_slot(4)

        beacon_nodes[1].connect(common_pb2.ConnectMessage(Address=addrs[0]))

        beacon_nodes[1].wait_for_slot(8)

        self.reset()


ex = Example()

ex.run()
