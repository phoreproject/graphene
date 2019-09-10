#below lines are required by every tests to import sibling package
import sys
sys.path.append('..')
sys.path.append('../pb')

from framework import tester
from framework import logger
from framework import beaconnode
from framework import validatornode
from framework import util
from framework import rpc
from framework import context
from framework import asserts
from framework import gotestrunner

class P2pInvalidRawData :
    def __init__(self) :
        logger.set_verbose(True)
        
    def run(self) :
        ctx = context.Context(
            directory = '/temp/synapse',
            delete_data_on_exit = not True
        )
        self._tester = tester.Tester(ctx)
        self._tester.run(self._do_run)
        
    def _do_run(self) :
        beacon_config = util.decode_json(util.read_file('data/regtest.json'))
        beacon_config['NetworkID'] = 'testnet'
        beacon_config['GenesisTime'] = 1566275808
        beacon_node_list = self._tester.create_nodes(
            1,
            node_class = beaconnode.BeaconNode,
            node_config_list = [ beacon_config ]
        )
        
        self._tester.start_all_nodes()
        util.sleep_for_seconds(10)
        
        peers = [
            '/ip4/0.0.0.0/tcp/11781/p2p/12D3KooWCnEau1w32qUwVDKDt9hH8H5889j7vmb2SYAoduhMDpmS',
        ]
        commands = [
            'invalidRawData'
        ]
        genesis = util.revert_hex_string(
            util.parse_genesis_hash(
                util.read_file(
                    beacon_node_list[0].get_stdout_log_file_name()
                )
            )
        )
        goTestRunner = gotestrunner.GoTestRunner('../../gotest', peers, commands, genesis)
        goTestRunner.run()
        
P2pInvalidRawData().run()
