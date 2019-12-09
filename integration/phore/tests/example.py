from phore.framework import tester, validatornode, shardnode


class Example(tester.Tester):
    def __init__(self):
        super().__init__()

    def _do_run(self):
        beacon_nodes = [self.create_beacon_node() for i in range(3)]
        beacon_nodes[0].start()
        beacon_nodes[0].wait_for_rpc()

        shard_node = self.create_shard_node(shardnode.ShardConfig.from_beacon(beacon_nodes[0]))
        shard_node.start()
        shard_node.wait_for_rpc()

        validator_node = self.create_validator_node(
            validatornode.ValidatorConfig.from_beacon_and_shard(beacon_nodes[0], shard_node, "0-255")
        )
        validator_node.start()
        validator_node.wait_for_rpc()

        beacon_nodes[0].wait_for_slot(17)


ex = Example()

ex.run()
