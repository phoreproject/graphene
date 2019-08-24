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

class RpcSubmitAttestation :
    def __init__(self) :
        logger.set_verbose(True)
        
    def run(self) :
        ctx = context.Context(
            directory = '/temp/synapse',
            delete_data_on_exit = False
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
        self.test_nil_attestation(rpc_client)
        self.test_invalid_data(rpc_client)
        
    def test_nil_attestation(self, rpc_client) :
        request = common_pb2.Attestation()
        try :
            response = rpc_client.SubmitAttestation(request)
        except Exception as e :
            asserts.assert_exception_contain_text(e, "attestation can't be nil")

    def test_invalid_data(self, rpc_client) :
        data = common_pb2.AttestationData()
        data.Slot = 1
        data.BeaconBlockHash = util.make_random_hash()
        data.TargetEpoch = 100
        data.TargetHash = util.make_random_hash()
        data.SourceEpoch = 100
        data.SourceHash = util.make_random_hash()
        data.ShardBlockHash = util.make_random_hash()
        data.Shard = 0
        data.LatestCrosslinkHash = util.make_random_hash()

        request = common_pb2.Attestation()
        request.Data.CopyFrom(data)
        try :
            response = rpc_client.SubmitAttestation(request)
            # this assert is not working yet
            #asserts.assert_not_here()
        except AssertionError :
            raise
        except Exception as e :
            asserts.assert_exception_contain_text(e, "attestation can't be nil")
        

RpcSubmitAttestation().run()
