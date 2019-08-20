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

class Example :
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
        beacon_count = 3
        beacon_listen_addresses = []
        for i in range(beacon_count) :
            beacon_listen_addresses.append(util.make_local_address(16000 + i))

        beacon_node_list = []
        
        beacon_config = util.decode_json(util.read_file('data/regtest.json'))
        beacon_config['NetworkID'] = 'testnet'
        beacon_config['GenesisTime'] = 0
        beacon_config_list = []
        
        # temp
        temp = [
            '/ip4/127.0.0.1/tcp/16000/ipfs/12D3KooWAgbLbupsbBiDBhPj8kpkkJSyzHtvfgrxKS53XyMLvW1J',
            '/ip4/127.0.0.1/tcp/16001/ipfs/12D3KooWQ6YrRqD1xbgdK6hBye5vTGuatJPJFZdD57mQMhD7egc8',
            '/ip4/127.0.0.1/tcp/16002/ipfs/12D3KooWJqp6cA184bfch1RNULuNYhEgicznrKc3WQRJuRU9z8Ys',
        ]
        for i in range(beacon_count) :
            beacon_config['listen'] = beacon_listen_addresses[i]
            peer_list = []
            for k in range(beacon_count) :
                if k != i :
                    #peer_list.append(beacon_listen_addresses[k])
                    if k < len(temp) :
                        peer_list.append(temp[k])
            beacon_config['BootstrapPeers'] = peer_list
            #beacon_config['connect'] = ','.join(peer_list)
            beacon_config_list.append(util.clone_dict(beacon_config))

        beacon_node_list = self._tester.create_nodes(
            beacon_count,
            node_class = beaconnode.BeaconNode,
            node_config_list = beacon_config_list
        )
            
        beacon_node_list[0].start()
        
        validator_config = {}
        validator_config['beaconhost'] = beacon_node_list[0].get_rpc_address()
        validtor_node_list = self._tester.create_nodes(
            1,
            node_class = validatornode.ValidatorNode,
            node_config_list = [ validator_config ]
        )
        validtor_node_list[0].start()

        util.sleep_for_seconds(10)

        beacon_node_list[1].start()
        util.sleep_for_seconds(10)

        #beacon_node_list[2].start()
        #util.sleep_for_seconds(10)

        rpc_client_list = []
        for beacon in beacon_node_list :
            rpc_client_list.append(rpc.create_beacon_rpc(beacon.get_rpc_address()))

        hash_list = [ None ] * beacon_count
        hash_list[0] = rpc_client_list[0].GetLastBlockHash(rpc.get_empty()).Hash
        hash_list[1] = rpc_client_list[1].GetLastBlockHash(rpc.get_empty()).Hash
        logger.info('Hash %d: %s' % (0, hash_list[0].hex()))
        logger.info('Hash %d: %s' % (1, hash_list[1].hex()))

        asserts.assert_true(hash_list[0] == hash_list[1])
        
Example().run()
