from . import util
from . import logger

class Node :
    def __init__(self) :
        self._context = None
        self._name = ''
        self._directory = ''
        self._config = None
        
    def initialize(
            self,
            context,
            name, # name can be the index of the node, such as 0, 1, 2
            directory, # the folder where node data is stored in, same as -datadir in Bitcoin
            config = None
        ) :
        self._context = context
        self._name = str(name)
        self._directory = directory
        self._config = config
        util.make_directories(self._directory)
        self.do_initialize()
    
    def get_context(self) :
        return self._context

    def get_config(self) :
        return self._config
        
    def get_directory(self) :
        return self._directory
        
    def get_name(self) :
        return self._name

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
        if self.do_start() :
            if wait :
                self.wait_for_start()
            logger.info('Node %s started' % (self.get_name()), verbose = True)
        else :
            logger.error('Node %s failed to start' % (self.get_name()))
        
    def stop(self) :
        if self.do_stop() :
            logger.info('Node %s stopped' % (self.get_name()), verbose = True)
        else :
            logger.error('Node %s failed to stop' % (self.get_name()))

    # Wait for the node starts completely
    def wait_for_start(self) :
        pass

    # derived class can override this
    def do_initialize(self) :
        pass
    
    def do_start(self) :
        return True
    
    def do_stop(self) :
        return True
