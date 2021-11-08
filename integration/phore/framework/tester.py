from phore.framework import beaconnode, shardnode, relayernode
from phore.framework import validatornode

import tempfile
import shutil
import os
import pathlib
import logging
import time


class Tester:
    def __init__(self):
        sh = logging.StreamHandler()
        sh.setFormatter(logging.Formatter('%(name)-12s: %(levelname)-8s %(message)s'))

        default_logger = logging.getLogger()
        default_logger.handlers = []
        default_logger.addHandler(sh)
        default_logger.setLevel(logging.DEBUG)

        self._node_list = []

        self._current_beacon_p2p_port = 16000
        self._current_beacon_rpc_port = 17000
        self._current_validator_rpc_port = 18000
        self._current_shard_rpc_port = 19000
        self._current_shard_p2p_port = 22000
        self._current_relayer_rpc_port = 20000
        self._current_relayer_p2p_port = 21000

        self._beacon_index = 1
        self._validator_index = 1
        self._shard_index = 1
        self._relayer_index = 1

        self.genesis_time = int(time.time()) + 5

        self._directory = tempfile.mkdtemp(suffix=None, prefix='graphene_test_', dir=None)

    def setup(self):
        logging.info('Test root directory: {}'.format(self._directory))

    def create_beacon_node(self, config: beaconnode.BeaconConfig = None) -> beaconnode.BeaconNode:
        # generate the config
        beacon_path = os.path.join(self._directory, "beacon{}".format(self._beacon_index))
        if config is None:
            config = beaconnode.BeaconConfig(self._beacon_index, beacon_path, self._current_beacon_p2p_port,
                                             self._current_beacon_rpc_port)

            config.genesis_time = self.genesis_time
            self._beacon_index += 1
            self._current_beacon_p2p_port += 1
            self._current_beacon_rpc_port += 1

        node = beaconnode.BeaconNode(config)

        self._node_list.append(node)

        return node

    def create_validator_node(self, config: validatornode.ValidatorConfig) -> validatornode.ValidatorNode:
        config.index = self._validator_index
        config.rpc_port = self._current_validator_rpc_port

        self._validator_index += 1
        self._current_validator_rpc_port += 1

        node = validatornode.ValidatorNode(config)

        self._node_list.append(node)

        return node

    def create_shard_node(self, config: shardnode.ShardConfig) -> shardnode.ShardNode:
        config.index = self._shard_index
        config.rpc_port = self._current_shard_rpc_port
        config.p2p_port = self._current_shard_p2p_port

        self._shard_index += 1
        self._current_shard_rpc_port += 1
        self._current_shard_p2p_port += 1

        node = shardnode.ShardNode(config)

        self._node_list.append(node)

        return node

    def create_relayer_node(self, config: relayernode.RelayerConfig) -> relayernode.RelayerNode:
        config.index = self._relayer_index
        config.p2p_port = self._current_relayer_p2p_port
        config.rpc_port = self._current_relayer_rpc_port

        self._relayer_index += 1
        self._current_relayer_p2p_port += 1
        self._current_relayer_rpc_port += 1

        node = relayernode.RelayerNode(config)

        self._node_list.append(node)

        return node

    def cleanup(self):
        self.stop_all_nodes()

        if os.path.isdir(self._directory):
            shutil.rmtree(self._directory)

    def _do_run(self):
        raise NotImplementedError("Implement _do_run for all tests")

    def run(self):
        self.setup()
        try:
            self._do_run()
        finally:
            self.cleanup()

    def stop_all_nodes(self):
        for node in self._node_list:
            node.stop()

    def reset(self):
        self.cleanup()

        self._node_list = []

        self._current_beacon_p2p_port = 16000
        self._current_beacon_rpc_port = 17000
        self._current_validator_rpc_port = 18000
        self._current_shard_rpc_port = 19000

        self._beacon_index = 1
        self._validator_index = 1
        self._shard_index = 1
        self._relayer_index = 1

        self.genesis_time = int(time.time()) + 5


def get_phore_path(*d):
    path = pathlib.Path(__file__)
    path = path.parent.parent.parent.parent.joinpath(*d)
    return str(path)
