# below lines are required by every tests to import sibling package
from framework import asserts
from framework import context
from framework import rpc
from framework import util
from framework import validatornode
from framework import beaconnode
from framework import logger
from framework import tester
import sys
sys.path.append('..')
sys.path.append('../pb')


class Example:
    def __init__(self):
        logger.set_verbose(True)

    def run(self):
        ctx = context.Context(
            directory='/tmp/synapse',
            delete_data_on_exit=not True
        )
        self._tester = tester.Tester(ctx)
        self._tester.run(self._do_run)

    def _do_run(self):
        beacon_count = 3
        beacon_listen_addresses = []
        for i in range(beacon_count):
            beacon_listen_addresses.append(util.make_local_address(16000 + i))

        beacon_node_list = []

        beacon_config = util.decode_json(util.read_file('data/regtest.json'))
        beacon_config['NetworkID'] = 'testnet'
        # Must hard code the GenesisTime, we can't use system time, because if one node is started in first run,
        # then another node is started in second run, they will get different GenesisTime, and cause different genesis hash
        beacon_config['GenesisTime'] = 1566275808
        beacon_config_list = []

        # temp
        temp = [
            '/ip4/127.0.0.1/tcp/16000/ipfs/12D3KooWGxKJ8fjCMZcUuB21bpybQv1nQsBSKtu5LhSrvJYGnFWj',
            '/ip4/127.0.0.1/tcp/16001/ipfs/12D3KooWPKur6Bn9Pk2W1NJGmurSzjAEDPr19JUgfDaqvNCGQauA',
            '/ip4/127.0.0.1/tcp/16002/ipfs/12D3KooWR5qZxxuoxAopeNte5Sbx32UD2oJnWtG9WpCkL6K9YFj3',
        ]
        for i in range(beacon_count):
            beacon_config['listen'] = beacon_listen_addresses[i]
            peer_list = []
            for k in range(beacon_count):
                if k != i:
                    # peer_list.append(beacon_listen_addresses[k])
                    if k < len(temp):
                        peer_list.append(temp[k])
            beacon_config['BootstrapPeers'] = peer_list
            #beacon_config['connect'] = ','.join(peer_list)
            beacon_config_list.append(util.clone_dict(beacon_config))

        beacon_node_list = self._tester.create_nodes(
            beacon_count,
            node_class=beaconnode.BeaconNode,
            node_config_list=beacon_config_list
        )

        rpc_client_list = []
        for beacon in beacon_node_list:
            rpc_client_list.append(rpc.create_beacon_rpc(beacon.get_rpc_address()))

        beacon_node_list[0].start()

        validator_config = {}
        validator_config['beaconhost'] = beacon_node_list[0].get_rpc_address()
        validtor_node_list = self._tester.create_nodes(
            1,
            node_class=validatornode.ValidatorNode,
            node_config_list=[validator_config]
        )
        validtor_node_list[0].start()

        util.sleep_for_seconds(10)

        beacon_node_list[1].start()
        util.sleep_for_seconds(10)

        previous_hash_list = [None] * beacon_count
        previous_hash_list[0] = rpc_client_list[0].GetLastBlockHash(rpc.get_empty()).Hash
        previous_hash_list[1] = rpc_client_list[1].GetLastBlockHash(rpc.get_empty()).Hash
        logger.info('Previous hash %d: %s' % (0, previous_hash_list[0].hex()))
        logger.info('Previous hash %d: %s' % (1, previous_hash_list[1].hex()))
        asserts.assert_true(previous_hash_list[0] == previous_hash_list[1])

        beacon_node_list[2].start()
        util.sleep_for_seconds(10)

        hash_list = [None] * beacon_count
        hash_list[0] = rpc_client_list[0].GetLastBlockHash(rpc.get_empty()).Hash
        hash_list[1] = rpc_client_list[1].GetLastBlockHash(rpc.get_empty()).Hash
        hash_list[2] = rpc_client_list[2].GetLastBlockHash(rpc.get_empty()).Hash
        logger.info('Hash %d: %s' % (0, hash_list[0].hex()))
        logger.info('Hash %d: %s' % (1, hash_list[1].hex()))
        logger.info('Hash %d: %s' % (2, hash_list[2].hex()))

        asserts.assert_true(hash_list[0] == hash_list[1])
        asserts.assert_true(hash_list[0] == hash_list[2])

        # Since new blocked were generated, these two hashes must not be same
        asserts.assert_true(hash_list[0] != previous_hash_list[0])


Example().run()
