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


class BasicBlockGeneration:
    def __init__(self):
        logger.set_verbose(True)

    def run(self):
        ctx = context.Context(
            #directory = '/tmp/synapse',
            #delete_data_on_exit = True
        )
        self._tester = tester.Tester(ctx)
        self._tester.run(self._do_run)

    def _do_run(self):
        beacon_config = util.decode_json(util.read_file('data/regtest.json'))
        beacon_config['NetworkID'] = 'testnet'
        beacon_config['GenesisTime'] = util.get_current_timestamp() + 5
        beacon_node_list = self._tester.create_nodes(
            1,
            node_class=beaconnode.BeaconNode,
            node_config_list=[beacon_config]
        )

        self._tester.start_all_nodes()
        util.sleep_for_seconds(5)

        rpc_client = rpc.create_beacon_rpc(beacon_node_list[0].get_rpc_address())

        logger.info('=====Test 1')
        logger.info('Validator is not running yet, no new block can be generated. The block hashes should be same.')

        first_hash, second_hash, third_hash = self._do_execute_and_get_hashes(rpc_client)
        asserts.assert_true(first_hash == second_hash)
        asserts.assert_true(first_hash == third_hash)
        asserts.assert_true(second_hash == third_hash)

        validator_config = {}
        validator_config['beaconhost'] = beacon_node_list[0].get_rpc_address()
        validtor_node_list = self._tester.create_nodes(
            1,
            node_class=validatornode.ValidatorNode,
            node_config_list=[validator_config]
        )
        validtor_node_list[0].start()

        logger.info('=====Test 2')
        logger.info('Validator is running, new blocks can be generated in a few seconds. The block hashes should not be same.')

        first_hash, second_hash, third_hash = self._do_execute_and_get_hashes(rpc_client)
        asserts.assert_true(first_hash != second_hash)
        asserts.assert_true(first_hash != third_hash)
        asserts.assert_true(second_hash != third_hash)

    def _do_execute_and_get_hashes(self, rpc_client):
        seconds_between_hashes = 15

        first_hash = rpc_client.GetLastBlockHash(rpc.get_empty()).Hash
        logger.info('1st block hash: %s' % (first_hash.hex()))

        util.sleep_for_seconds(seconds_between_hashes)

        second_hash = rpc_client.GetLastBlockHash(rpc.get_empty()).Hash
        logger.info('2nd block hash: %s' % (second_hash.hex()))

        util.sleep_for_seconds(seconds_between_hashes)

        third_hash = rpc_client.GetLastBlockHash(rpc.get_empty()).Hash
        logger.info('3rd block hash: %s' % (third_hash.hex()))

        return (first_hash, second_hash, third_hash)


BasicBlockGeneration().run()
