from .node import Node
from . import util
from .process import Process

import os

class ValidatorNode(Node):
    def __init__(self):
        Node.__init__(self)
        self._process = None

    def do_start(self):
        self._stdout_file_name = os.path.join(self.get_directory(), 'validator.stdout.log')
        node_config = self.get_config()
        self._process = Process(stdout = self._stdout_file_name).run(
            self.get_context().get_validator_executable(),
            '-beaconhost',
            util.get_dict_value(node_config, 'beaconhost', ':11782'),
            '-networkid',
            util.get_dict_value(node_config, 'networkid', 'testnet'),
            '-rootkey',
            util.get_dict_value(node_config, 'rootkey', 'testnet'),
            '-validators',
            '0-255'
        )
        return True

    def do_stop(self):
        if self._process != None:
            self._process.kill()
            self._process = None
        return True

