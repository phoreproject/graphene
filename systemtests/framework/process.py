import subprocess

class Process :
    @staticmethod
    def run(*args) :
        p = Process()
        p._do_run(*args)
        return p

    def __init__(self) :
        self._process = None

    def _do_run(self, *args) :
        self._process = subprocess.Popen(
            args,
            stdin = subprocess.PIPE,
            stdout = subprocess.PIPE,
            stderr = subprocess.STDOUT,
            universal_newlines = True,
            bufsize = 0
        )

    def wait_until_finish(self) :
        return self._process.communicate()
        
    def kill(self) :
        if self._process != None :
            self._process.kill()
        self._process = None
    
    # !!!Will block if nothing to read
    def read_stdout(self) :
        return self._process.stdout.readline()

