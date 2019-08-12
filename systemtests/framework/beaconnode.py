from .node import Node
from . import util
from .process import Process

class BeaconNode(Node) :
    def __init__(self) :
        Node.__init__(self)
        self._process = None

    def do_start(self) :
        self._process = Process(capture_stdout = False).run(
            self.get_context().get_beacon_executable(),
            '-chainconfig',
            '../../regtest.json'
        )
        return True
    
    def do_stop(self) :
        if self._process != None :
            self._process.kill()
        return True
    
