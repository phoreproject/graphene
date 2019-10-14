# below lines are required by every tests to import sibling package
from pb import common_pb2
from pb import rpc_pb2
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


class RpcSubmitBlockRequest:
    def __init__(self):
        logger.set_verbose(True)

    def run(self):
        ctx = context.Context(
            #directory = '/tmp/synapse',
            #delete_data_on_exit = False
        )
        self._tester = tester.Tester(ctx)
        self._tester.run(self._do_run)

    def _do_run(self):
        beacon_config = util.decode_json(util.read_file('data/regtest.json'))
        beacon_config['NetworkID'] = 'testnet'
        beacon_config['GenesisTime'] = 1566275808
        beacon_node_list = self._tester.create_nodes(
            1,
            node_class=beaconnode.BeaconNode,
            node_config_list=[beacon_config]
        )

        self._tester.start_all_nodes()
        util.sleep_for_seconds(5)

        rpc_client = rpc.create_beacon_rpc(beacon_node_list[0].get_rpc_address())
        self.test_nil_block(rpc_client)
        self.test_nil_header(rpc_client)
        self.test_nil_body(rpc_client)
        self.test_no_parent_block(rpc_client)
        self.test_has_parent_block(rpc_client)

    def test_nil_block(self, rpc_client):
        request = rpc_pb2.SubmitBlockRequest()
        try:
            response = rpc_client.SubmitBlock(request)
        except Exception as e:
            asserts.assert_exception_contain_text(e, "block can't be nil")

    def test_nil_header(self, rpc_client):
        block = common_pb2.Block()

        request = rpc_pb2.SubmitBlockRequest()
        request.Block.CopyFrom(block)
        try:
            response = rpc_client.SubmitBlock(request)
        except Exception as e:
            asserts.assert_exception_contain_text(e, "block header can't be nil")

    def test_nil_body(self, rpc_client):
        header = common_pb2.BlockHeader()
        header.SlotNumber = 1
        header.ParentRoot = util.make_random_hash()
        header.StateRoot = util.make_random_hash()
        header.RandaoReveal = util.make_random_hash()
        header.Signature = util.make_random_hash()

        block = common_pb2.Block()
        block.Header.CopyFrom(header)

        request = rpc_pb2.SubmitBlockRequest()
        request.Block.CopyFrom(block)
        try:
            response = rpc_client.SubmitBlock(request)
        except Exception as e:
            asserts.assert_exception_contain_text(e, "block body can't be nil")

    def test_no_parent_block(self, rpc_client):
        header = common_pb2.BlockHeader()
        header.SlotNumber = 1
        header.ParentRoot = util.make_random_hash()
        header.StateRoot = util.make_random_hash()
        header.RandaoReveal = util.make_random_hash()
        header.Signature = util.make_random_hash()

        body = common_pb2.BlockBody()

        block = common_pb2.Block()
        block.Header.CopyFrom(header)
        block.Body.CopyFrom(body)

        request = rpc_pb2.SubmitBlockRequest()
        request.Block.CopyFrom(block)
        try:
            response = rpc_client.SubmitBlock(request)
        except Exception as e:
            asserts.assert_exception_contain_text(e, "do not have parent block")

    def test_has_parent_block(self, rpc_client):
        blockHashRequest = rpc_pb2.GetBlockHashRequest()
        blockHashRequest.SlotNumber = 0
        genesisHash = rpc_client.GetBlockHash(blockHashRequest).Hash

        header = common_pb2.BlockHeader()
        header.SlotNumber = 1
        header.ParentRoot = genesisHash
        header.StateRoot = util.make_random_hash()
        header.RandaoReveal = util.make_random_hash()
        header.Signature = util.make_random_hash()

        body = common_pb2.BlockBody()

        block = common_pb2.Block()
        block.Header.CopyFrom(header)
        block.Body.CopyFrom(body)

        request = rpc_pb2.SubmitBlockRequest()
        request.Block.CopyFrom(block)
        try:
            response = rpc_client.SubmitBlock(request)
            asserts.assert_not_here()
        # Need to catch AssertionError if there is assert in the try block
        except AssertionError:
            raise
        except Exception as e:
            # The exception is 'unexpected compression mode' or 'unexpected information in compressed infinity'
            # not sure where it's from yet
            pass


RpcSubmitBlockRequest().run()
