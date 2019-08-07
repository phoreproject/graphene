from . import util
from . import logger

class Node :
    def __init__(
            self,
            name, # name can be the index of the node, such as 0, 1, 2
            directory, # the folder where node data is stored in, same as -datadir in Bitcoin
            config = None
        ) :
        self._name = str(name)
        self._directory = directory
        self._config = config
        util.make_directories(self._directory)

    # execute a command via RPC or CLI
    # for example, execute('setgenerate', True, 10)
    def execute(self, *args) :
        normalized_args = []
        for arg in args :
            if type(arg) == bool :
                if arg == True :
                    arg = 'true'
                else :
                    arg = 'false'
            else :
                arg = str(arg)
            normalized_args.append(arg)
        print('Execute: %s' % (str(normalized_args)))

    def get_name(self) :
        return self._name

    def start(self, wait = True) :
        if wait :
            self.wait_for_start()
        logger.info('Node %s started' % (self.get_name()), verbose = True)
        
    def stop(self) :
        logger.info('Node %s stopped' % (self.get_name()), verbose = True)

    # Wait for the node starts completely
    def wait_for_start(self) :
        pass
