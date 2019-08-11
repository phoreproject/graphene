from .node import Node
from . import util
from .process import Process

class ValidatorNode(Node) :
    def __init__(self) :
        Node.__init__(self)
        self._process = None

    def do_start(self) :
        self._process = Process.run(
            self.get_context().get_validator_executable(),
            '-rootkey',
            'testnet',
            '-validators',
            '0-255'
        )
        return True
    
    def do_stop(self) :
        if self._process != None :
            self._process.kill()
        return True
    
