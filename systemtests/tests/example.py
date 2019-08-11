#below two lines are required by every tests to import sibling package
import sys
sys.path.append('..')

from framework import tester
from framework import logger
from framework import beaconnode
from framework import validatornode
from framework import util

class Example :
    def __init__(self) :
        logger.set_verbose(True)
        
    def run(self) :
        self._tester = tester.Tester()
        self._tester.run(self._do_run)
        
    def _do_run(self) :
        self._tester.create_nodes(1, node_class = beaconnode.BeaconNode)
        #self._tester.create_nodes(1, node_class = validatornode.ValidatorNode)
        self._tester.start_all_nodes()
        self._tester.get_node(0).execute('setgenerate', True, 10)
        util.sleep_for_seconds(2)
        self._tester.stop_all_nodes()

Example().run()
