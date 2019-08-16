from .node import Node
from . import util
from .process import Process

import os

class BeaconNode(Node) :
    def __init__(self) :
        Node.__init__(self)
        self._process = None

    def get_rpc_port(self) :
        return 15000 + self.get_index()
        
    def get_rpc_address(self) :
        return '127.0.0.1:%d' % (self.get_rpc_port())
        
    def do_start(self) :
        self._config_file_name = os.path.join(self.get_directory(), 'testconfig.json')
        self._create_config_file(self._config_file_name)

        self._stdout_file_name = os.path.join(self.get_directory(), 'beacon.stdout.log')
        
        self._process = Process(stdout = self._stdout_file_name).run(
            self.get_context().get_beacon_executable(),
            '-datadir',
            self.get_directory(),
            '-rpclisten',
            self.get_rpc_address(),
            '-chainconfig',
            self._config_file_name
        )
        return True
    
    def do_stop(self) :
        if self._process != None :
            self._process.kill()
            self._process = None
        return True
    
    def _create_config_file(self, file_name) :
        config = {}
        
        node_config = self.get_config()
        config['GenesisTime'] = util.get_dict_value(node_config, 'GenesisTime', 0)
        config['NetworkID'] = util.get_dict_value(node_config, 'NetworkID', 'regtest')
        config['BootstrapPeers'] = util.get_dict_value(node_config, 'BootstrapPeers', [])
        config['InitialValidators'] = util.get_dict_value(node_config, 'InitialValidators', {})
        
        util.write_file(file_name, util.encode_json(config))
