from .node import Node
from . import util
from . import logger
from .context import Context

import tempfile 
import shutil
import os

class Tester :
    def __init__(
            self,
            context = None,
        ) :
        self._context = context
        if self._context == None :
            self._context = Context()
        print(self._context.get_beacon_executable())
        
        self._node_list = []
        self._node_name_map = {}
        self._directory = self._context.get_directory()
        if self._directory == None :
            self._directory = tempfile.mkdtemp(suffix = None, prefix = 'synapse_test_', dir = None)

    def setup(self) :
        util.make_directories(self._directory)
        logger.info('Test root directory: %s' % (self._directory))
        
    def cleanup(self) :
        if self._context.should_delete_data_on_exit() :
            shutil.rmtree(self._directory)
            
    def run(self, runner) :
        self.setup()
        try :
            runner()
        finally :
            self.cleanup()

    def start_all_nodes(self) :
        for node in self._node_list :
            node.start()

    def stop_all_nodes(self) :
        for node in self._node_list :
            node.stop()

    def create_nodes(
            self,
            count,
            node_class,
            node_config = None,
            node_names = None # if node_names is None, each node has name as its index (0, 1, 2, etc)
        ) :
        result_list = []
        for i in range(count) :
            name = ''
            if node_names == None or i >= len(node_names) :
                name = str(len(self._node_list))
            else :
                name = str(node_names[i])
            
            path = os.path.join(self._directory, name)
            node = node_class()
            node.initialize(self._context, name, path, node_config)
            self._node_list.append(node)
            
            assert name not in self._node_name_map
            self._node_name_map[name] = node
            
            result_list.append(node)
        return result_list

    def get_node_list(self) :
        return self._node_list

    def get_node(self, name) :
        return util.get_dict_value(self._node_name_map, str(name))
