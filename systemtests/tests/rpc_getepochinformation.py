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

from pb import rpc_pb2
from pb import common_pb2

class RpcGetEpochInformation :
    def __init__(self) :
        logger.set_verbose(True)
        
    def run(self) :
        ctx = context.Context(
            #directory = '/temp/synapse',
            #delete_data_on_exit = False
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
        util.sleep_for_seconds(5)

        rpc_client = rpc.create_beacon_rpc(beacon_node_list[0].get_rpc_address())
        #self.test_invalid_request(rpc_client)
        #self.test_invalid_request_bug(rpc_client)
        
    def test_invalid_request(self, rpc_client) :
        asserts.assert_bug('GetEpochInformation should return error?')

        request = rpc_pb2.EpochInformationRequest()
        request.EpochIndex = 10
        try :
            response = rpc_client.GetEpochInformation(request)
            asserts.assert_not_here()
        except Exception as e :
            asserts.assert_exception_contain_text(e, "don't have state for block hash")

    def test_invalid_request_bug(self, rpc_client) :
        asserts.assert_bug('Setting EpochIndex to 10000000 causes the program uses all RAM and crashes the OS.')

        request = rpc_pb2.EpochInformationRequest()
        request.EpochIndex = 10000000
        try :
            response = rpc_client.GetEpochInformation(request)
        except Exception as e :
            asserts.assert_exception_contain_text(e, "don't have state for block hash")


RpcGetEpochInformation().run()
