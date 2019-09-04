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

import asyncio

import multiaddr

from libp2p import new_node
from libp2p.network.stream.net_stream_interface import INetStream
from libp2p.peer.peerinfo import info_from_p2p_addr
from libp2p.typing import TProtocol

PROTOCOL_ID = TProtocol("/grpc/phore/0.0.1")

def cancel_tasks() :
    tasks = asyncio.Task.all_tasks()           
    for t in tasks:
        t.cancel()

async def read_data(stream: INetStream) -> None:
    while True:
        read_bytes = await stream.read()
        if read_bytes is not None:
            read_string = read_bytes.decode()
            if read_string != "\n":
                print("%s " % read_string, end="")


async def write_data(stream: INetStream) -> None:
    await stream.write('aaaaaaaaaaaaaaa')

async def _do_run_it(self) :    
    # "/ip4/127.0.0.1/tcp/8000/p2p/QmQn4SwGkDZKkUEpBRBvTmheQycxAHJUNmVEnjA2v1qe8Q"
    #destination = '/ip4/0.0.0.0/tcp/11781/ipfs/12D3KooWSzvx4mxBSGhLzqS3gLp4wrGqivLuNghZadTgH1R1fbVR'
    destination = '/ip4/0.0.0.0/tcp/11781/p2p/12D3KooWSzvx4mxBSGhLzqS3gLp4wrGqivLuNghZadTgH1R1fbVR'
    maddr = multiaddr.Multiaddr(destination)
    info = info_from_p2p_addr(maddr)

    transport_opt = "/ip4/127.0.0.1/tcp/30000"
    host = await new_node(transport_opt=[transport_opt])

    # Associate the peer with local ip address
    await host.connect(info)

    # Start a stream with the destination.
    # Multiaddress of the destination peer is fetched from the peerstore using 'peerId'.
    stream = await host.new_stream(info.peer_id, [PROTOCOL_ID])

    #asyncio.ensure_future(read_data(stream))
    #asyncio.ensure_future(write_data(stream))
    await write_data(stream)
    util.sleep_for_seconds(10)
    

class P2pInvalidData :
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
        
        f = asyncio.ensure_future(_do_run_it(self))
        loop = asyncio.get_event_loop()
        try:
            loop.run_until_complete(f)
            #loop.run_forever()
        except KeyboardInterrupt:
            pass
        finally:
            cancel_tasks()
            loop.close()

P2pInvalidData().run()
